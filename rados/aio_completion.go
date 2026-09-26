//go:build ceph_preview

package rados

/*
#cgo LDFLAGS: -lrados
#include <stdint.h>
#include <rados/librados.h>

extern void aioCompleteCallback(uintptr_t);

static inline void aio_callback_complete(rados_completion_t c, void *arg) {
	aioCompleteCallback((uintptr_t)arg);
}

static inline int aio_create_callback_completion(uintptr_t id,
	rados_completion_t *pc) {
	return rados_aio_create_completion2((void *)id, aio_callback_complete, pc);
}
*/
import "C"

import (
	"runtime"

	"github.com/ceph/go-ceph/internal/callbacks"
	"github.com/ceph/go-ceph/internal/log"
)

// aioCompletions maps the integer ids handed to librados as callback
// arguments to their AioCompletion.
var aioCompletions = callbacks.New()

// AioCompletion tracks a read or write operation started by OperateAsync.
//
// The completion owns everything librados may still use while the
// operation is in flight: the operation's steps and the Go buffers they
// handed to librados, which stay pinned. Release must be called once the
// completion is no longer needed.
type AioCompletion struct {
	c    C.rados_completion_t
	id   uintptr
	kind opKind
	op   operation

	pinner runtime.Pinner
	done   chan struct{}

	ret     int
	version uint64
	err     error
}

// operateAsync creates a completion, moves the steps of o into it, and
// calls submit to start the operation. If submit fails, the steps are
// returned to o.
func operateAsync(
	kind opKind, o *operation, submit func(C.rados_completion_t) C.int) (*AioCompletion, error) {

	c := &AioCompletion{
		kind: kind,
		done: make(chan struct{}),
	}
	c.pin(o.steps)
	c.id = aioCompletions.Add(c)

	var cc C.rados_completion_t
	if ret := C.aio_create_callback_completion(C.uintptr_t(c.id), &cc); ret < 0 {
		aioCompletions.Remove(c.id)
		c.pinner.Unpin()
		return nil, getError(ret)
	}
	c.c = cc
	c.op.steps, o.steps = o.steps, nil

	if ret := submit(c.c); ret < 0 {
		o.steps, c.op.steps = c.op.steps, nil
		aioCompletions.Remove(c.id)
		C.rados_aio_release(c.c)
		c.c = nil
		c.pinner.Unpin()
		return nil, getError(ret)
	}
	runtime.SetFinalizer(c, aioCompletionFinalizer)
	return c, nil
}

// pin pins every Go buffer that a step handed to librados, so that it
// stays valid until Release.
func (c *AioCompletion) pin(steps []opStep) {
	pinBytes := func(b []byte) {
		if len(b) > 0 {
			c.pinner.Pin(&b[0])
		}
	}
	for _, s := range steps {
		switch s := s.(type) {
		case *readStep:
			pinBytes(s.b)
		case *writeStep:
			pinBytes(s.b)
		}
	}
}

//export aioCompleteCallback
func aioCompleteCallback(id C.uintptr_t) {
	aioComplete(uintptr(id))
}

// aioComplete records the result of the operation identified by id, runs
// the update of each of its steps, and closes its done channel.
func aioComplete(id uintptr) {
	c, ok := aioCompletions.Lookup(id).(*AioCompletion)
	if !ok {
		log.Warnf("rados: completion for unknown id %d", id)
		return
	}
	aioCompletions.Remove(id)

	ret := C.rados_aio_get_return_value(c.c)
	c.ret = int(ret)
	c.version = uint64(C.rados_aio_get_version(c.c))
	if c.kind == readOp {
		ret = readOpResult(ret)
	}
	c.err = c.op.update(c.kind, ret)
	close(c.done)
}

// Done returns a channel that is closed when librados reports that the
// operation has completed.
func (c *AioCompletion) Done() <-chan struct{} {
	return c.done
}

// ReturnValue returns the return value of the operation. It is valid only
// after the channel returned by Done is closed. A read operation guarded by
// a true CmpXattr returns 1, which Err does not report as an error.
//
// Implements:
//
//	int rados_aio_get_return_value(rados_completion_t c);
func (c *AioCompletion) ReturnValue() int {
	return c.ret
}

// Err returns the error Operate would have returned for the operation:
// nil on success, otherwise an OperationError built from the return value
// and the errors of the operation's steps. It is valid only after the
// channel returned by Done is closed.
func (c *AioCompletion) Err() error {
	return c.err
}

// Version returns the version of the object after the operation. It is
// valid only after the channel returned by Done is closed.
//
// Implements:
//
//	uint64_t rados_aio_get_version(rados_completion_t c);
func (c *AioCompletion) Version() uint64 {
	return c.version
}

// Release waits for the operation to complete, then frees the C completion,
// unpins the buffers the operation used and frees its steps' C memory. The
// results the steps copied into Go memory stay readable. Release may be
// called more than once, but not concurrently.
//
// Implements:
//
//	void rados_aio_release(rados_completion_t c);
func (c *AioCompletion) Release() {
	<-c.done
	if c.c == nil {
		return
	}
	runtime.SetFinalizer(c, nil)
	c.release()
}

func (c *AioCompletion) release() {
	C.rados_aio_release(c.c)
	c.c = nil
	c.pinner.Unpin()
	c.op.free()
}

func aioCompletionFinalizer(c *AioCompletion) {
	log.Warnf("unreleased AioCompletion found. Cleaning up.")
	c.release()
}
