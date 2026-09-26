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

// writeOpOmapCmpStep holds the C-allocated result of an OmapCmp action.
type writeOpOmapCmpStep struct {
	prval *C.int
}

func newWriteOpOmapCmpStep() *writeOpOmapCmpStep {
	s := &writeOpOmapCmpStep{
		prval: (*C.int)(C.malloc(C.sizeof_int)),
	}
	*s.prval = 0
	return s
}

func (s *writeOpOmapCmpStep) update() error {
	return getError(*s.prval)
}

func (s *writeOpOmapCmpStep) free() {
	C.free(unsafe.Pointer(s.prval))
	s.prval = nil
}

// OmapCmp asserts that the omap value stored under key compares to value
// as op specifies; if it does not, the whole write operation fails with
// ECANCELED and none of its actions are applied. A missing key compares as
// an empty value. The stored value is the left-hand operand: CmpXattrOpLt
// holds when the stored value is less than value. The OSD supports only
// CmpXattrOpEq, CmpXattrOpLt and CmpXattrOpGt here; other operators fail
// the operation with EINVAL.
//
// The key is passed with its length, so it may contain NUL bytes.
//
// Implements:
//
//	void rados_write_op_omap_cmp2(rados_write_op_t write_op,
//	                              const char *key,
//	                              uint8_t comparison_operator,
//	                              const char *val,
//	                              size_t key_len,
//	                              size_t val_len,
//	                              int *prval);
func (w *WriteOp) OmapCmp(key string, op CmpXattrOp, value []byte) {
	s := newWriteOpOmapCmpStep()
	w.steps = append(w.steps, s)

	// C.CString copies every byte of key, including any NUL bytes.
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	// librados copies the key and value before returning.
	var cValue *C.char
	if len(value) > 0 {
		cValue = (*C.char)(unsafe.Pointer(&value[0]))
	}

	C.rados_write_op_omap_cmp2(
		w.op,
		cKey,
		C.uint8_t(op),
		cValue,
		C.size_t(len(key)),
		C.size_t(len(value)),
		s.prval)
}
