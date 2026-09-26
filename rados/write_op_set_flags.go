//go:build ceph_preview

package rados

// #cgo LDFLAGS: -lrados
// #include <rados/librados.h>
//
import "C"

// SetFlags sets per-op flags on the action most recently added to the write
// operation, for example OpFlagFailOk to let the operation succeed even if
// that action fails. The flags are LIBRADOS_OP_FLAG_* values (OpFlags), not
// the OperationFlags passed to Operate.
//
// Implements:
//
//	void rados_write_op_set_flags(rados_write_op_t write_op,
//	                              int flags);
func (w *WriteOp) SetFlags(flags OpFlags) {
	C.rados_write_op_set_flags(w.op, C.int(flags))
}
