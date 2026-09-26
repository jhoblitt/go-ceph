//go:build ceph_preview

package rados

// #cgo LDFLAGS: -lrados
// #include <rados/librados.h>
//
import "C"

// Truncate sets the size of the object to offset, discarding or
// zero-extending its data as needed.
//
// Implements:
//
//	void rados_write_op_truncate(rados_write_op_t write_op,
//	                             uint64_t offset);
func (w *WriteOp) Truncate(offset uint64) {
	C.rados_write_op_truncate(w.op, C.uint64_t(offset))
}
