//go:build ceph_preview

package rados

import (
	"syscall"

	"github.com/stretchr/testify/assert"
)

func (suite *RadosTestSuite) TestWriteOpOmapCmp() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.SetOmap(oid, map[string][]byte{"key": []byte("value")}))

	writeIf := func(key string, op CmpXattrOp, v []byte, data string) error {
		wop := CreateWriteOp()
		defer wop.Release()
		wop.OmapCmp(key, op, v)
		wop.WriteFull([]byte(data))
		return wop.Operate(suite.ioctx, oid, OperationNoFlag)
	}
	read := func() string {
		buf := make([]byte, 64)
		n, err := suite.ioctx.Read(oid, buf, 0)
		ta.NoError(err)
		return string(buf[:n])
	}
	canceled := -int(syscall.ECANCELED)

	// a mismatch fails the whole operation
	ta.Equal(canceled, opErrorCode(writeIf("key", CmpXattrOpEq, []byte("other"), "first")))
	ta.Equal("", read())
	// an empty value is compared, not dereferenced
	ta.Equal(canceled, opErrorCode(writeIf("key", CmpXattrOpEq, nil, "first")))
	ta.Equal("", read())
	// a missing key compares as an empty value
	ta.NoError(writeIf("missing", CmpXattrOpEq, nil, "second"))
	ta.Equal("second", read())

	ta.NoError(writeIf("key", CmpXattrOpEq, []byte("value"), "third"))
	ta.Equal("third", read())
	// the stored value is the left-hand operand: "value" < "zzzzz"
	ta.NoError(writeIf("key", CmpXattrOpLt, []byte("zzzzz"), "fourth"))
	ta.Equal("fourth", read())

	// keys are passed with their length, so a NUL byte is part of the key
	ta.NoError(suite.ioctx.SetOmap(oid, map[string][]byte{"k\x00ey": []byte("nul")}))
	ta.Equal(canceled, opErrorCode(writeIf("k\x00ey", CmpXattrOpEq, []byte("value"), "fifth")))
	ta.Equal("fourth", read())
	ta.NoError(writeIf("k\x00ey", CmpXattrOpEq, []byte("nul"), "sixth"))
	ta.Equal("sixth", read())
}
