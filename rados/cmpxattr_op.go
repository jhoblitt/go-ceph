//go:build ceph_preview

package rados

// #cgo LDFLAGS: -lrados
// #include <rados/librados.h>
//
import "C"

// CmpXattrOp is the comparison operator used by the CmpXattr and OmapCmp
// actions of read and write operations.
type CmpXattrOp int

const (
	// CmpXattrOpEq compares for equality.
	CmpXattrOpEq = CmpXattrOp(C.LIBRADOS_CMPXATTR_OP_EQ)
	// CmpXattrOpNe compares for inequality.
	CmpXattrOpNe = CmpXattrOp(C.LIBRADOS_CMPXATTR_OP_NE)
	// CmpXattrOpGt compares for greater than.
	CmpXattrOpGt = CmpXattrOp(C.LIBRADOS_CMPXATTR_OP_GT)
	// CmpXattrOpGte compares for greater than or equal.
	CmpXattrOpGte = CmpXattrOp(C.LIBRADOS_CMPXATTR_OP_GTE)
	// CmpXattrOpLt compares for less than.
	CmpXattrOpLt = CmpXattrOp(C.LIBRADOS_CMPXATTR_OP_LT)
	// CmpXattrOpLte compares for less than or equal.
	CmpXattrOpLte = CmpXattrOp(C.LIBRADOS_CMPXATTR_OP_LTE)
)
