//go:build ceph_preview

package rados

import (
	"github.com/stretchr/testify/assert"
)

func (suite *RadosTestSuite) TestWriteOpTruncate() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.WriteFull(oid, []byte("0123456789")))

	wop := CreateWriteOp()
	defer wop.Release()
	wop.Truncate(4)
	ta.NoError(wop.Operate(suite.ioctx, oid, OperationNoFlag))

	stat, err := suite.ioctx.Stat(oid)
	ta.NoError(err)
	ta.Equal(uint64(4), stat.Size)
}
