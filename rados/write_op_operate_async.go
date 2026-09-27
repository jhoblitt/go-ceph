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

// OperateAsync starts the operation(s) and returns without waiting for
// them. The returned AioCompletion reports when the operation is done and
// takes over the op's steps: they are updated before the completion's Done
// channel is closed, and freed by AioCompletion.Release rather than by
// WriteOp.Release, which may be called as soon as OperateAsync returns. The
// op cannot be operated again.
//
// Implements:
//
//	int rados_aio_write_op_operate(rados_write_op_t write_op,
//	                               rados_ioctx_t io,
//	                               rados_completion_t completion,
//	                               const char *oid,
//	                               time_t *mtime,
//	                               int flags);
func (w *WriteOp) OperateAsync(ioctx *IOContext, oid string, flags OperationFlags) (*AioCompletion, error) {
	if err := ioctx.validate(); err != nil {
		return nil, err
	}

	cOid := C.CString(oid)
	defer C.free(unsafe.Pointer(cOid))

	return operateAsync(writeOp, &w.operation, func(c C.rados_completion_t) C.int {
		return C.rados_aio_write_op_operate(w.op, ioctx.ioctx, c, cOid, nil, C.int(flags))
	})
}
