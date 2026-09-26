//go:build ceph_preview

package rados

// #cgo LDFLAGS: -lrados
// #include <rados/librados.h>
//
import "C"

// Zero overwrites length bytes of the object, starting at offset, with
// zeros.
//
// Implements:
//
//	void rados_write_op_zero(rados_write_op_t write_op,
//	                         uint64_t offset,
//	                         uint64_t len);
func (w *WriteOp) Zero(offset, length uint64) {
	C.rados_write_op_zero(w.op, C.uint64_t(offset), C.uint64_t(length))
}
