//go:build ceph_preview

package rados

import (
	"github.com/stretchr/testify/assert"
)

// A class method's positive return value reaches a read operation only
// with OperationReturnVec; like C++ librados, the op must count it as
// success.
func (suite *RadosTestSuite) TestReadOpPositiveReturn() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.SetOmap(oid, map[string][]byte{"key": []byte("value")}))

	rop := CreateReadOp()
	defer rop.Release()
	// cls_hello's write_return_data returns 42
	es := rop.Exec("hello", "write_return_data", nil)
	gos := rop.GetOmapValues("", "", 10)
	ta.NoError(rop.Operate(suite.ioctx, oid, OperationReturnVec))

	out, err := es.Bytes()
	ta.NoError(err)
	ta.Equal([]byte("you might see this"), out)
	kv, err := gos.Next()
	ta.NoError(err)
	ta.Equal(&OmapKeyValue{Key: "key", Value: []byte("value")}, kv)
}
