package rados

// #include <stdint.h>
import "C"

import (
	"runtime"
	"unsafe"
)

type readStep struct {
	withoutUpdate
	// the c pointer utilizes the Go byteslice data, which is pinned because
	// librados keeps the pointer until the operation completes

	// inputs:
	b []byte

	// arguments:
	cBuffer  *C.char
	cReadLen C.size_t
	cOffset  C.uint64_t

	pinner runtime.Pinner
}

func newReadStep(b []byte, offset uint64) *readStep {
	rs := &readStep{
		b:        b,
		cReadLen: C.size_t(len(b)),
		cOffset:  C.uint64_t(offset),
	}
	if len(b) > 0 {
		rs.cBuffer = (*C.char)(unsafe.Pointer(&b[0]))
		rs.pinner.Pin(rs.cBuffer)
	}
	// An unreleased op must not leave the buffer pinned: the runtime panics
	// when it collects a Pinner that still holds pinned memory.
	runtime.SetFinalizer(rs, opStepFinalizer)
	return rs
}

func (rs *readStep) free() {
	rs.pinner.Unpin()
	runtime.SetFinalizer(rs, nil)
}
