//go:build !(pacific || quincy) && ceph_preview

package rados

import (
	"syscall"

	"github.com/stretchr/testify/assert"
)

func (suite *RadosTestSuite) TestReadOpCmpXattrStat() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	data := []byte("guarded data")
	ta.NoError(suite.ioctx.WriteFull(oid, data))
	ta.NoError(suite.ioctx.SetXattr(oid, "attr", []byte("value")))
	stat, err := suite.ioctx.Stat(oid)
	ta.NoError(err)

	guardedStat := func(v []byte) (*ReadOpStatStep, error) {
		rop := CreateReadOp()
		defer rop.Release()
		rop.CmpXattr("attr", CmpXattrOpEq, v)
		s := rop.Stat()
		return s, rop.Operate(suite.ioctx, oid, OperationNoFlag)
	}

	// a true guard lets the following Stat report the object
	s, err := guardedStat([]byte("value"))
	ta.NoError(err)
	ta.Equal(uint64(len(data)), s.Size())
	ta.Equal(stat.ModTime.Unix(), s.ModTime().Unix())

	// a false guard fails the operation
	_, err = guardedStat([]byte("other"))
	ta.Equal(-int(syscall.ECANCELED), opErrorCode(err))
}
