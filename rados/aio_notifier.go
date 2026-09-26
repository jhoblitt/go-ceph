//go:build ceph_preview

package rados

/*
#cgo LDFLAGS: -lrados
#include <errno.h>
#include <stdint.h>
#include <unistd.h>
#include <rados/librados.h>

extern void aioCompleteCallback(uintptr_t);

// aio_pipe_fd is the write end of the notification pipe. It only changes
// while no pipe-mode completion is in flight.
static int aio_pipe_fd = -1;

static void aio_set_pipe_fd(int fd) {
	__atomic_store_n(&aio_pipe_fd, fd, __ATOMIC_RELEASE);
}

// aio_pipe_complete writes the completion's 8-byte id to the pipe. A write
// of at most PIPE_BUF bytes is atomic, so ids never interleave. Should the
// write fail, it completes the operation through the Go callback instead,
// so that no completion is lost.
static void aio_pipe_complete(rados_completion_t c, void *arg) {
	uint64_t id = (uint64_t)(uintptr_t)arg;
	const char *p = (const char *)&id;
	size_t left = sizeof(id);
	int fd = __atomic_load_n(&aio_pipe_fd, __ATOMIC_ACQUIRE);

	while (left > 0) {
		ssize_t n = write(fd, p, left);
		if (n < 0) {
			if (errno == EINTR) {
				continue;
			}
			aioCompleteCallback((uintptr_t)arg);
			return;
		}
		p += n;
		left -= (size_t)n;
	}
}

static int aio_create_pipe_completion(uintptr_t id, rados_completion_t *pc) {
	return rados_aio_create_completion2((void *)id, aio_pipe_complete, pc);
}
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/ceph/go-ceph/internal/log"
)

// AioMode selects how librados' completion notifications reach Go.
type AioMode int

const (
	// AioModeCallback completes each operation in a cgo callback on the
	// librados thread that reports it. It is the default.
	AioModeCallback AioMode = iota
	// AioModePipe has the librados thread write the completion's id to a
	// pipe and completes the operation on a single goroutine that drains
	// the pipe, so no librados thread calls into Go.
	AioModePipe
)

// aioNotifier is the process-wide notifier state. OperateAsync holds mu
// for reading while it starts an operation; SetAioMode holds it for
// writing.
var aioNotifier struct {
	mu   sync.RWMutex
	pipe *aioPipe // nil in AioModeCallback
	err  error    // set when the selected mode cannot be used
}

// aioPipe is the notification pipe of AioModePipe and its drain goroutine.
type aioPipe struct {
	r, w *os.File

	// inflight counts the pipe-mode completions not yet completed.
	inflight sync.WaitGroup

	start   sync.Once
	started atomic.Bool
	stopped chan struct{} // closed when the drain goroutine exits
}

// SetAioMode selects how the completions of operations started by
// OperateAsync afterwards are delivered. The setting is process-wide and is
// meant to be made once, before the first OperateAsync.
//
// In AioModePipe a single goroutine, started by the first OperateAsync,
// drains the pipe. Switching away from AioModePipe waits until every
// operation started in that mode has completed, then closes the pipe, which
// stops the goroutine. Otherwise the goroutine and the pipe live until the
// process exits.
//
// If the pipe cannot be created, or m is not a known mode, OperateAsync
// returns the error until SetAioMode is called again.
func SetAioMode(m AioMode) {
	aioNotifier.mu.Lock()
	defer aioNotifier.mu.Unlock()

	if aioNotifier.pipe != nil {
		if m == AioModePipe {
			return
		}
		aioNotifier.pipe.stop()
		aioNotifier.pipe = nil
	}
	aioNotifier.err = nil

	switch m {
	case AioModeCallback:
	case AioModePipe:
		p, err := newAioPipe()
		if err != nil {
			aioNotifier.err = err
			return
		}
		aioNotifier.pipe = p
	default:
		aioNotifier.err = fmt.Errorf("rados: invalid AioMode %d", m)
	}
}

func newAioPipe() (*aioPipe, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("rados: creating the completion pipe: %w", err)
	}
	p := &aioPipe{
		r:       r,
		w:       w,
		stopped: make(chan struct{}),
	}
	// Fd puts w in blocking mode, which is what the C writer expects.
	C.aio_set_pipe_fd(C.int(w.Fd()))
	return p, nil
}

// createCompletion creates a pipe-mode completion for id and starts the
// drain goroutine if it is not running yet. The caller must hold
// aioNotifier.mu for reading, and call completed once the operation has
// been completed or has failed to start.
func (p *aioPipe) createCompletion(id uintptr) (C.rados_completion_t, error) {
	p.start.Do(func() {
		p.started.Store(true)
		go p.drain()
	})
	p.inflight.Add(1)

	var cc C.rados_completion_t
	if ret := C.aio_create_pipe_completion(C.uintptr_t(id), &cc); ret < 0 {
		p.completed()
		return nil, getError(ret)
	}
	return cc, nil
}

func (p *aioPipe) completed() {
	p.inflight.Done()
}

// drain completes the operation of each id read from the pipe. It returns
// when the write end of the pipe is closed.
func (p *aioPipe) drain() {
	defer close(p.stopped)
	var buf [8]byte
	for {
		if _, err := io.ReadFull(p.r, buf[:]); err != nil {
			return
		}
		aioComplete(uintptr(binary.NativeEndian.Uint64(buf[:])))
	}
}

// stop waits for every in-flight pipe-mode completion, then closes the pipe
// and waits for the drain goroutine to exit. The caller must hold
// aioNotifier.mu for writing.
func (p *aioPipe) stop() {
	p.inflight.Wait()
	C.aio_set_pipe_fd(-1)
	if err := p.w.Close(); err != nil {
		log.Warnf("rados: closing the completion pipe: %v", err)
	}
	if p.started.Load() {
		<-p.stopped
	}
	if err := p.r.Close(); err != nil {
		log.Warnf("rados: closing the completion pipe: %v", err)
	}
}
