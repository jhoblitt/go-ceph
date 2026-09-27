//go:build !(pacific || quincy) && ceph_preview

package rados

// #cgo LDFLAGS: -lrados
// #include <rados/librados.h>
// #include <stdlib.h>
// #include <time.h>
//
import "C"

import (
	"time"
	"unsafe"

	ts "github.com/ceph/go-ceph/internal/timespec"
)

// ReadOpStatStep holds the result of the Stat read operation.
// Result is valid only after Operate() was called.
type ReadOpStatStep struct {
	// C returned data:
	psize  *C.uint64_t
	pmtime *C.struct_timespec
	prval  *C.int

	size    uint64
	modTime time.Time
}

func newReadOpStatStep() *ReadOpStatStep {
	s := &ReadOpStatStep{
		psize:  (*C.uint64_t)(C.malloc(C.sizeof_uint64_t)),
		pmtime: (*C.struct_timespec)(C.malloc(C.sizeof_struct_timespec)),
		prval:  (*C.int)(C.malloc(C.sizeof_int)),
	}
	*s.psize = 0
	*s.pmtime = C.struct_timespec{}
	*s.prval = 0
	return s
}

func (s *ReadOpStatStep) update() error {
	// A positive prval is not an error: a true CmpXattr earlier in the
	// operation leaves its result, 1, in the prval of later actions.
	if err := getErrorIfNegative(*s.prval); err != nil {
		return err
	}
	s.size = uint64(*s.psize)
	t := ts.CStructToTimespec(ts.CTimespecPtr(s.pmtime))
	s.modTime = time.Unix(t.Sec, t.Nsec)
	return nil
}

func (s *ReadOpStatStep) free() {
	C.free(unsafe.Pointer(s.psize))
	C.free(unsafe.Pointer(s.pmtime))
	C.free(unsafe.Pointer(s.prval))
	s.psize = nil
	s.pmtime = nil
	s.prval = nil
}

// Size returns the size of the object in bytes. It is zero until the
// operation has been performed successfully.
func (s *ReadOpStatStep) Size() uint64 {
	return s.size
}

// ModTime returns the modification time of the object. It is the zero
// time until the operation has been performed successfully.
func (s *ReadOpStatStep) ModTime() time.Time {
	return s.modTime
}

// Stat fetches the size and modification time of the object as part of
// the read operation. It needs Ceph Reef or later.
//
// Implements:
//
//	void rados_read_op_stat2(rados_read_op_t read_op,
//	                         uint64_t *psize,
//	                         struct timespec *pmtime,
//	                         int *prval);
func (r *ReadOp) Stat() *ReadOpStatStep {
	s := newReadOpStatStep()
	r.steps = append(r.steps, s)
	C.rados_read_op_stat2(r.op, s.psize, s.pmtime, s.prval)
	return s
}
