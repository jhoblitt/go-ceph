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

// Append appends b to the end of the object.
//
// Implements:
//
//	void rados_write_op_append(rados_write_op_t write_op,
//	                           const char *buffer,
//	                           size_t len);
func (w *WriteOp) Append(b []byte) {
	// librados copies the buffer before returning.
	var cBuffer *C.char
	if len(b) > 0 {
		cBuffer = (*C.char)(unsafe.Pointer(&b[0]))
	}

	C.rados_write_op_append(w.op, cBuffer, C.size_t(len(b)))
}
