//go:build ceph_preview

package rados

import (
	"github.com/stretchr/testify/assert"
)

func (suite *RadosTestSuite) TestWriteOpAppend() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.WriteFull(oid, []byte("abc")))

	appendBytes := func(b []byte) {
		wop := CreateWriteOp()
		defer wop.Release()
		wop.Append(b)
		ta.NoError(wop.Operate(suite.ioctx, oid, OperationNoFlag))
	}
	read := func() string {
		buf := make([]byte, 64)
		n, err := suite.ioctx.Read(oid, buf, 0)
		ta.NoError(err)
		return string(buf[:n])
	}

	appendBytes([]byte("def"))
	ta.Equal("abcdef", read())
	// an empty append is a no-op, not a dereference
	appendBytes(nil)
	appendBytes([]byte{})
	ta.Equal("abcdef", read())
}
