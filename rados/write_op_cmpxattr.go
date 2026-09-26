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

// CmpXattr asserts that the xattr name of the object compares to value as
// op specifies; if it does not, the whole write operation fails with
// ECANCELED and none of its actions are applied. The comparison is a
// byte-wise string comparison with value as the left-hand operand:
// CmpXattrOpGt holds when value is greater than the stored xattr.
//
// Implements:
//
//	void rados_write_op_cmpxattr(rados_write_op_t write_op,
//	                             const char *name,
//	                             uint8_t comparison_operator,
//	                             const char *value,
//	                             size_t value_len);
func (w *WriteOp) CmpXattr(name string, op CmpXattrOp, value []byte) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	// librados copies the value before returning.
	var cValue *C.char
	if len(value) > 0 {
		cValue = (*C.char)(unsafe.Pointer(&value[0]))
	}

	C.rados_write_op_cmpxattr(
		w.op,
		cName,
		C.uint8_t(op),
		cValue,
		C.size_t(len(value)))
}
