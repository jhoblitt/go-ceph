//go:build !(pacific || quincy) && ceph_preview

package rados

import (
	"time"

	"github.com/stretchr/testify/assert"
)

func (suite *RadosTestSuite) TestReadOpStat() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	data := []byte("stat this object")
	ta.NoError(suite.ioctx.WriteFull(oid, data))

	stat, err := suite.ioctx.Stat(oid)
	ta.NoError(err)

	rop := CreateReadOp()
	defer rop.Release()
	step := rop.Stat()
	err = rop.Operate(suite.ioctx, oid, OperationNoFlag)
	ta.NoError(err)
	ta.Equal(uint64(len(data)), step.Size())
	// rados_stat reports whole seconds; stat2 carries nanoseconds.
	ta.Equal(stat.ModTime.Unix(), step.ModTime().Unix())
	ta.WithinDuration(time.Now(), step.ModTime(), 10*time.Minute)

	// the object does not exist
	rop2 := CreateReadOp()
	defer rop2.Release()
	step2 := rop2.Stat()
	err = rop2.Operate(suite.ioctx, suite.GenObjectName(), OperationNoFlag)
	ta.ErrorIs(err, ErrNotFound)
	ta.Equal(uint64(0), step2.Size())
	ta.True(step2.ModTime().IsZero())
}
