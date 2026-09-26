//go:build ceph_preview

package rados

import (
	"syscall"

	"github.com/stretchr/testify/assert"
)

func (suite *RadosTestSuite) TestWriteOpCmpXattr() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	name := "attr"
	value := []byte("value")
	ta.NoError(suite.ioctx.SetXattr(oid, name, value))

	writeIf := func(op CmpXattrOp, v []byte, data string) error {
		wop := CreateWriteOp()
		defer wop.Release()
		wop.CmpXattr(name, op, v)
		wop.WriteFull([]byte(data))
		return wop.Operate(suite.ioctx, oid, OperationNoFlag)
	}
	read := func() string {
		buf := make([]byte, 64)
		n, err := suite.ioctx.Read(oid, buf, 0)
		ta.NoError(err)
		return string(buf[:n])
	}

	// a mismatch fails the whole operation
	err := writeIf(CmpXattrOpEq, []byte("other"), "first")
	ta.Equal(-int(syscall.ECANCELED), opErrorCode(err))
	ta.Equal("", read())
	// an empty value is compared, not dereferenced
	err = writeIf(CmpXattrOpEq, nil, "first")
	ta.Equal(-int(syscall.ECANCELED), opErrorCode(err))
	ta.Equal("", read())

	ta.NoError(writeIf(CmpXattrOpEq, value, "second"))
	ta.Equal("second", read())
	// the supplied value is the left-hand operand: "zzzzz" > "value"
	ta.NoError(writeIf(CmpXattrOpGt, []byte("zzzzz"), "third"))
	ta.Equal("third", read())
}
