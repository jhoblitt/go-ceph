//go:build ceph_preview

package rados

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *RadosTestSuite) TestReadOpOperateAsync() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	data := []byte("read asynchronously")
	ta.NoError(suite.ioctx.WriteFull(oid, data))
	objv := encodeObjVersion(7, "async")
	wop := CreateWriteOp()
	wop.Exec("version", "set", objv)
	ta.NoError(wop.Operate(suite.ioctx, oid, OperationNoFlag))
	wop.Release()

	rop := CreateReadOp()
	buf := make([]byte, 64)
	readStep := rop.Read(0, buf)
	execStep := rop.Exec("version", "read", nil)
	c, err := rop.OperateAsync(suite.ioctx, oid, OperationNoFlag)
	require.NoError(suite.T(), err)
	rop.Release()
	<-c.Done()
	ta.Equal(0, c.ReturnValue())
	ta.NoError(c.Err())
	ta.Equal(int64(len(data)), readStep.BytesRead)
	ta.Equal(data, buf[:readStep.BytesRead])
	c.Release()
	// the exec output outlives both the op and the completion
	out, err := execStep.Bytes()
	ta.NoError(err)
	ta.Equal(objv, out)

	// a missing object fails the operation
	rop2 := CreateReadOp()
	defer rop2.Release()
	rop2.Read(0, buf)
	c2, err := rop2.OperateAsync(suite.ioctx, suite.GenObjectName(), OperationNoFlag)
	require.NoError(suite.T(), err)
	defer c2.Release()
	<-c2.Done()
	ta.Less(c2.ReturnValue(), 0)
	ta.ErrorIs(c2.Err(), ErrNotFound)
}
