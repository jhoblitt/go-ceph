//go:build ceph_preview

package rados

// #cgo LDFLAGS: -lrados
// #include <rados/librados.h>
// #include <stdlib.h>
//
import "C"

import (
	"unsafe"
)

// ReadOpGetXattrsStep holds the result of the GetXattrs read operation.
// Result is valid only after Operate() was called.
type ReadOpGetXattrsStep struct {
	// C returned data:
	iter  C.rados_xattrs_iter_t
	prval *C.int

	// Go copies of the xattrs, filled in by update.
	xattrs     map[string][]byte
	canReadAll bool
}

func newReadOpGetXattrsStep() *ReadOpGetXattrsStep {
	s := &ReadOpGetXattrsStep{
		prval: (*C.int)(C.malloc(C.sizeof_int)),
	}
	*s.prval = 0
	return s
}

func (s *ReadOpGetXattrsStep) update() error {
	// A positive prval is not an error: a true CmpXattr earlier in the
	// operation leaves its result, 1, in the prval of later actions.
	if err := getErrorIfNegative(*s.prval); err != nil {
		return err
	}
	xattrs := make(map[string][]byte)
	for {
		var (
			cName *C.char
			cVal  *C.char
			cLen  C.size_t
		)
		ret := C.rados_getxattrs_next(s.iter, &cName, &cVal, &cLen)
		if ret < 0 {
			return getError(ret)
		}
		if cName == nil {
			break
		}
		xattrs[C.GoString(cName)] = C.GoBytes(unsafe.Pointer(cVal), C.int(cLen))
	}
	s.xattrs = xattrs
	s.canReadAll = true
	return nil
}

func (s *ReadOpGetXattrsStep) free() {
	if s.iter != nil {
		C.rados_getxattrs_end(s.iter)
		s.iter = nil
	}
	C.free(unsafe.Pointer(s.prval))
	s.prval = nil
}

// Xattrs returns the extended attributes of the object, keyed by name.
// May be called only after Operate() finished successfully.
func (s *ReadOpGetXattrsStep) Xattrs() (map[string][]byte, error) {
	if !s.canReadAll {
		return nil, ErrOperationIncomplete
	}
	return s.xattrs, nil
}

// GetXattrs fetches all extended attributes of the object as part of the
// read operation. The returned step provides the attributes after the
// operation has been performed.
//
// Implements:
//
//	void rados_read_op_getxattrs(rados_read_op_t read_op,
//	                             rados_xattrs_iter_t *iter,
//	                             int *prval);
func (r *ReadOp) GetXattrs() *ReadOpGetXattrsStep {
	s := newReadOpGetXattrsStep()
	r.steps = append(r.steps, s)
	C.rados_read_op_getxattrs(r.op, &s.iter, s.prval)
	return s
}
