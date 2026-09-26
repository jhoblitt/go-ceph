//go:build ceph_preview

package rados

// #cgo LDFLAGS: -lrados
// #include <rados/librados.h>
// #include <stdlib.h>
//
import "C"

import (
	"fmt"
	"strings"
	"unsafe"
)

// ReadOpOmapGetKeysStep holds the result of the GetOmapKeys read operation.
// Result is valid only after Operate() was called.
type ReadOpOmapGetKeysStep struct {
	// C returned data:
	iter  C.rados_omap_iter_t
	pmore *C.uchar
	prval *C.int

	// Go copies of the results, filled in by update.
	keys    []string
	more    bool
	canRead bool

	// err holds a validation error detected before the operation is
	// performed.
	err error
}

func newReadOpOmapGetKeysStep() *ReadOpOmapGetKeysStep {
	s := &ReadOpOmapGetKeysStep{
		pmore: (*C.uchar)(C.malloc(C.sizeof_uchar)),
		prval: (*C.int)(C.malloc(C.sizeof_int)),
	}
	*s.pmore = 0
	*s.prval = 0
	return s
}

func (s *ReadOpOmapGetKeysStep) update() error {
	if s.err != nil {
		return s.err
	}
	// A positive prval is not an error: a true CmpXattr earlier in the
	// operation leaves its result, 1, in the prval of later actions.
	if err := getErrorIfNegative(*s.prval); err != nil {
		return err
	}
	keys := []string{}
	for {
		var (
			cKey    *C.char
			cVal    *C.char
			cKeyLen C.size_t
			cValLen C.size_t
		)
		ret := C.rados_omap_get_next2(s.iter, &cKey, &cVal, &cKeyLen, &cValLen)
		if ret != 0 {
			return getError(ret)
		}
		if cKey == nil {
			break
		}
		keys = append(keys, string(C.GoBytes(unsafe.Pointer(cKey), C.int(cKeyLen))))
	}
	s.keys = keys
	s.more = *s.pmore != 0
	s.canRead = true
	return nil
}

func (s *ReadOpOmapGetKeysStep) free() {
	if s.iter != nil {
		C.rados_omap_get_end(s.iter)
		s.iter = nil
	}
	C.free(unsafe.Pointer(s.pmore))
	C.free(unsafe.Pointer(s.prval))
	s.pmore = nil
	s.prval = nil
}

// Keys returns the omap keys, in order. May be called only after Operate()
// finished successfully.
func (s *ReadOpOmapGetKeysStep) Keys() ([]string, error) {
	if !s.canRead {
		return nil, ErrOperationIncomplete
	}
	return s.keys, nil
}

// More reports whether keys beyond those returned by Keys are available.
func (s *ReadOpOmapGetKeysStep) More() bool {
	return s.more
}

// GetOmapKeys lists up to maxReturn omap keys of the object that sort after
// startAfter, as part of the read operation.
//
// startAfter is passed to librados as a NUL-terminated C string and must
// not contain a NUL byte; if it does, Operate returns an error wrapping
// ErrNulInString.
//
// Implements:
//
//	void rados_read_op_omap_get_keys2(rados_read_op_t read_op,
//	                                  const char *start_after,
//	                                  uint64_t max_return,
//	                                  rados_omap_iter_t *iter,
//	                                  unsigned char *pmore,
//	                                  int *prval);
func (r *ReadOp) GetOmapKeys(startAfter string, maxReturn uint64) *ReadOpOmapGetKeysStep {
	s := newReadOpOmapGetKeysStep()
	r.steps = append(r.steps, s)

	if strings.IndexByte(startAfter, 0) >= 0 {
		s.err = fmt.Errorf("startAfter: %w", ErrNulInString)
		return s
	}

	cStartAfter := C.CString(startAfter)
	defer C.free(unsafe.Pointer(cStartAfter))

	C.rados_read_op_omap_get_keys2(
		r.op,
		cStartAfter,
		C.uint64_t(maxReturn),
		&s.iter,
		s.pmore,
		s.prval,
	)
	return s
}
