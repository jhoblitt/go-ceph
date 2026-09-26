//go:build ceph_preview

package rados

import (
	"github.com/stretchr/testify/assert"
)

func (suite *RadosTestSuite) TestWriteOpSetFlags() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.SetXattr(oid, "attr", []byte("value")))

	writeIf := func(flags OpFlags, data string) error {
		wop := CreateWriteOp()
		defer wop.Release()
		wop.CmpXattr("attr", CmpXattrOpEq, []byte("other"))
		if flags != 0 {
			wop.SetFlags(flags)
		}
		wop.WriteFull([]byte(data))
		return wop.Operate(suite.ioctx, oid, OperationNoFlag)
	}
	read := func() string {
		buf := make([]byte, 64)
		n, err := suite.ioctx.Read(oid, buf, 0)
		ta.NoError(err)
		return string(buf[:n])
	}

	ta.Error(writeIf(0, "first"))
	ta.Equal("", read())

	// FAILOK on the comparison lets the rest of the operation proceed
	ta.NoError(writeIf(OpFlagFailOk, "second"))
	ta.Equal("second", read())
}
