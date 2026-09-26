//go:build !(pacific || quincy) && ceph_preview

package rados

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// operateAsyncWithMtime writes data to oid with OperateAsyncWithMtime and
// waits for the completion.
func (suite *RadosTestSuite) operateAsyncWithMtime(oid string, mtime Timespec) {
	wop := CreateWriteOp()
	wop.WriteFull([]byte("stamped"))
	c, err := wop.OperateAsyncWithMtime(suite.ioctx, oid, mtime, OperationNoFlag)
	require.NoError(suite.T(), err)
	// the completion, not the op, owns what librados still needs
	wop.Release()
	<-c.Done()
	assert.NoError(suite.T(), c.Err())
	c.Release()
}

func (suite *RadosTestSuite) TestWriteOpOperateAsyncWithMtime() {
	suite.SetupConnection()
	suite.forEachAioMode(func() {
		ta := assert.New(suite.T())

		oid := suite.GenObjectName()
		mtime := Timespec{Sec: 1600000000, Nsec: 123456789}
		suite.operateAsyncWithMtime(oid, mtime)

		stat, err := suite.ioctx.Stat(oid)
		ta.NoError(err)
		ta.Equal(uint64(len("stamped")), stat.Size)
		// rados_stat reports whole seconds
		ta.Equal(time.Unix(mtime.Sec, 0), stat.ModTime)
	})
}

func (suite *RadosTestSuite) TestWriteOpOperateAsyncWithMtimeNsec() {
	suite.SetupConnection()
	suite.forEachAioMode(func() {
		ta := assert.New(suite.T())

		oid := suite.GenObjectName()
		mtime := Timespec{Sec: 1600000000, Nsec: 123456789}
		suite.operateAsyncWithMtime(oid, mtime)

		rop := CreateReadOp()
		defer rop.Release()
		step := rop.Stat()
		ta.NoError(rop.Operate(suite.ioctx, oid, OperationNoFlag))
		ta.Equal(time.Unix(mtime.Sec, mtime.Nsec), step.ModTime())
	})
}
