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

// RmXattr removes the xattr name from the object.
//
// Implements:
//
//	void rados_write_op_rmxattr(rados_write_op_t write_op,
//	                            const char *name);
func (w *WriteOp) RmXattr(name string) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	C.rados_write_op_rmxattr(w.op, cName)
}
