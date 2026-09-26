//go:build ceph_preview

package rados

import (
	"github.com/stretchr/testify/assert"
)

func (suite *RadosTestSuite) TestReadOpGetXattrs() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	xattrs := map[string][]byte{
		"attr1": []byte("value1"),
		"attr2": []byte("value2"),
	}

	wop := CreateWriteOp()
	defer wop.Release()
	wop.Create(CreateIdempotent)
	for name, value := range xattrs {
		wop.SetXattr(name, value)
	}
	err := wop.Operate(suite.ioctx, oid, OperationNoFlag)
	ta.NoError(err)

	rop := CreateReadOp()
	defer rop.Release()
	step := rop.GetXattrs()

	_, err = step.Xattrs()
	ta.ErrorIs(err, ErrOperationIncomplete)

	err = rop.Operate(suite.ioctx, oid, OperationNoFlag)
	ta.NoError(err)
	got, err := step.Xattrs()
	ta.NoError(err)
	ta.Equal(xattrs, got)

	// the object has no xattrs: an empty map, not an error
	oid2 := suite.GenObjectName()
	ta.NoError(suite.ioctx.Create(oid2, CreateIdempotent))
	rop2 := CreateReadOp()
	defer rop2.Release()
	step2 := rop2.GetXattrs()
	err = rop2.Operate(suite.ioctx, oid2, OperationNoFlag)
	ta.NoError(err)
	got, err = step2.Xattrs()
	ta.NoError(err)
	ta.Empty(got)

	// the object does not exist
	rop3 := CreateReadOp()
	defer rop3.Release()
	step3 := rop3.GetXattrs()
	err = rop3.Operate(suite.ioctx, suite.GenObjectName(), OperationNoFlag)
	ta.ErrorIs(err, ErrNotFound)
	_, err = step3.Xattrs()
	ta.Error(err)
}

func (suite *RadosTestSuite) TestReadOpGetXattrsAsync() {
	suite.SetupConnection()
	suite.forEachAioMode(func() {
		ta := assert.New(suite.T())

		oid := suite.GenObjectName()
		ta.NoError(suite.ioctx.SetXattr(oid, "attr", []byte("value")))

		// a true guard in front must not hide the xattrs
		rop := CreateReadOp()
		rop.CmpXattr("attr", CmpXattrOpEq, []byte("value"))
		step := rop.GetXattrs()
		c, err := rop.OperateAsync(suite.ioctx, oid, OperationNoFlag)
		ta.NoError(err)
		rop.Release()
		<-c.Done()
		ta.NoError(c.Err())
		c.Release()
		got, err := step.Xattrs()
		ta.NoError(err)
		ta.Equal(map[string][]byte{"attr": []byte("value")}, got)
	})
}
