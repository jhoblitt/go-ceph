//go:build ceph_preview

package rados

import (
	"github.com/stretchr/testify/assert"
)

func (suite *RadosTestSuite) TestWriteOpRmXattr() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.SetXattr(oid, "keep", []byte("1")))
	ta.NoError(suite.ioctx.SetXattr(oid, "drop", []byte("2")))

	wop := CreateWriteOp()
	defer wop.Release()
	wop.RmXattr("drop")
	ta.NoError(wop.Operate(suite.ioctx, oid, OperationNoFlag))

	xattrs, err := suite.ioctx.ListXattrs(oid)
	ta.NoError(err)
	ta.Equal(map[string][]byte{"keep": []byte("1")}, xattrs)
}
