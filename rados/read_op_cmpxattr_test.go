//go:build ceph_preview

package rados

import (
	"errors"
	"syscall"

	"github.com/stretchr/testify/assert"
)

// opErrorCode returns the return code of a failed Operate call, or 0 when
// err is nil or carries no code.
func opErrorCode(err error) int {
	var oe OperationError
	if !errors.As(err, &oe) {
		return 0
	}
	var ec interface{ ErrorCode() int }
	if !errors.As(oe.OpError, &ec) {
		return 0
	}
	return ec.ErrorCode()
}

func (suite *RadosTestSuite) TestReadOpCmpXattr() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	name := "attr"
	value := []byte("value")
	data := []byte("guarded data")
	ta.NoError(suite.ioctx.WriteFull(oid, data))
	ta.NoError(suite.ioctx.SetXattr(oid, name, value))

	cmp := func(op CmpXattrOp, v []byte) error {
		rop := CreateReadOp()
		defer rop.Release()
		rop.CmpXattr(name, op, v)
		return rop.Operate(suite.ioctx, oid, OperationNoFlag)
	}
	canceled := -int(syscall.ECANCELED)

	ta.NoError(cmp(CmpXattrOpEq, value))
	ta.NoError(cmp(CmpXattrOpNe, []byte("other")))
	// the supplied value is the left-hand operand: "zzzzz" > "value"
	ta.NoError(cmp(CmpXattrOpGt, []byte("zzzzz")))
	ta.NoError(cmp(CmpXattrOpLte, value))

	// a false comparison fails the whole operation
	ta.Equal(canceled, opErrorCode(cmp(CmpXattrOpEq, []byte("other"))))
	ta.Equal(canceled, opErrorCode(cmp(CmpXattrOpLt, value)))
	// an empty value is compared, not dereferenced
	ta.Equal(canceled, opErrorCode(cmp(CmpXattrOpEq, nil)))
	ta.Equal(canceled, opErrorCode(cmp(CmpXattrOpEq, []byte{})))

	// the actions after a true guard run and report their results
	rop := CreateReadOp()
	defer rop.Release()
	rop.CmpXattr(name, CmpXattrOpEq, value)
	buf := make([]byte, 64)
	readStep := rop.Read(0, buf)
	ta.NoError(rop.Operate(suite.ioctx, oid, OperationNoFlag))
	ta.Equal(data, buf[:readStep.BytesRead])

	// omap reads after a true guard return their data
	ta.NoError(suite.ioctx.SetOmap(oid, map[string][]byte{"key": []byte("omap")}))
	rop2 := CreateReadOp()
	defer rop2.Release()
	rop2.CmpXattr(name, CmpXattrOpEq, value)
	gos := rop2.GetOmapValues("", "", 10)
	byKeys := rop2.GetOmapValuesByKeys([]string{"key"})
	ta.NoError(rop2.Operate(suite.ioctx, oid, OperationNoFlag))
	want := &OmapKeyValue{Key: "key", Value: []byte("omap")}
	kv, err := gos.Next()
	ta.NoError(err)
	ta.Equal(want, kv)
	kv, err = byKeys.Next()
	ta.NoError(err)
	ta.Equal(want, kv)
}
