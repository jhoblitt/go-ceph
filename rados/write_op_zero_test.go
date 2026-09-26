//go:build ceph_preview

package rados

import (
	"github.com/stretchr/testify/assert"
)

func (suite *RadosTestSuite) TestWriteOpZero() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.WriteFull(oid, []byte("abcdef")))

	wop := CreateWriteOp()
	defer wop.Release()
	wop.Zero(1, 2)
	ta.NoError(wop.Operate(suite.ioctx, oid, OperationNoFlag))

	buf := make([]byte, 64)
	n, err := suite.ioctx.Read(oid, buf, 0)
	ta.NoError(err)
	ta.Equal([]byte("a\x00\x00def"), buf[:n])
}
