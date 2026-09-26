//go:build !(pacific || quincy) && ceph_preview

package rados

// #cgo LDFLAGS: -lrados
// #include <rados/librados.h>
// #include <stdlib.h>
// #include <time.h>
//
import "C"

import (
	"unsafe"

	ts "github.com/ceph/go-ceph/internal/timespec"
)

// OperateAsyncWithMtime starts the operation(s) like OperateAsync, setting
// the object's modification time to mtime instead of the current time.
// librados copies mtime before the call returns. It needs Ceph Reef or
// later.
//
// Implements:
//
//	int rados_aio_write_op_operate2(rados_write_op_t write_op,
//	                                rados_ioctx_t io,
//	                                rados_completion_t completion,
//	                                const char *oid,
//	                                struct timespec *mtime,
//	                                int flags);
func (w *WriteOp) OperateAsyncWithMtime(
	ioctx *IOContext, oid string, mtime Timespec, flags OperationFlags) (*AioCompletion, error) {

	if err := ioctx.validate(); err != nil {
		return nil, err
	}

	cOid := C.CString(oid)
	defer C.free(unsafe.Pointer(cOid))
	cMtime := &C.struct_timespec{}
	ts.CopyToCStruct(ts.Timespec(mtime), ts.CTimespecPtr(cMtime))

	return operateAsync(writeOp, &w.operation, func(c C.rados_completion_t) C.int {
		return C.rados_aio_write_op_operate2(w.op, ioctx.ioctx, c, cOid, cMtime, C.int(flags))
	})
}
