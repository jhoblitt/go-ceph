//go:build ceph_preview

package rados

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *RadosTestSuite) TestWriteOpOperateAsync() {
	suite.SetupConnection()
	suite.forEachAioMode(func() {
		ta := assert.New(suite.T())

		oid := suite.GenObjectName()
		data := []byte("written asynchronously")

		wop := CreateWriteOp()
		wop.WriteFull(data)
		wop.SetXattr("attr", []byte("value"))
		c, err := wop.OperateAsync(suite.ioctx, oid, OperationNoFlag)
		require.NoError(suite.T(), err)
		// the completion, not the op, owns what librados still needs
		wop.Release()
		<-c.Done()
		ta.Equal(0, c.ReturnValue())
		ta.NoError(c.Err())
		ta.NotZero(c.Version())
		c.Release()
		c.Release() // idempotent

		buf := make([]byte, 64)
		n, err := suite.ioctx.Read(oid, buf, 0)
		ta.NoError(err)
		ta.Equal(data, buf[:n])

		// a failing guard fails the operation
		wop2 := CreateWriteOp()
		defer wop2.Release()
		wop2.CmpXattr("attr", CmpXattrOpEq, []byte("other"))
		wop2.Exec("version", "set", encodeObjVersion(1, "async"))
		c2, err := wop2.OperateAsync(suite.ioctx, oid, OperationNoFlag)
		require.NoError(suite.T(), err)
		defer c2.Release()
		<-c2.Done()
		ta.Less(c2.ReturnValue(), 0)
		ta.Error(c2.Err())
	})
}
