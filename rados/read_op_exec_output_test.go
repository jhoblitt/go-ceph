//go:build ceph_preview

package rados

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadOpExecStepOutputIncomplete(t *testing.T) {
	rdOp := CreateReadOp()
	defer rdOp.Release()
	es := rdOp.Exec("hello", "say_hello", nil)
	out, err := es.Output()
	assert.ErrorIs(t, err, ErrOperationIncomplete)
	assert.Nil(t, out)
}

func (suite *RadosTestSuite) TestReadOpExecOutput() {
	suite.SetupConnection()

	// cls_hello's write_return_data, given any input, puts "too much input
	// data!" in its output and returns -EINVAL. The OSD sends a failed op's
	// output back, and OperationReturnVec keeps it in the reply to a resent
	// op too.
	suite.T().Run("failed call", func(t *testing.T) {
		ta := assert.New(t)
		rdOp := CreateReadOp()
		defer rdOp.Release()
		es := rdOp.Exec("hello", "write_return_data", []byte("x"))
		err := rdOp.Operate(suite.ioctx, suite.GenObjectName(), OperationReturnVec)
		ta.Equal(-int(syscall.EINVAL), opErrorCode(err))

		out, err := es.Output()
		require.NoError(t, err)
		ta.Equal([]byte("too much input data!"), out)
		out, err = es.Bytes()
		ta.ErrorIs(err, ErrOperationIncomplete)
		ta.Nil(out)
	})

	suite.T().Run("successful call", func(t *testing.T) {
		ta := assert.New(t)
		oid := suite.GenObjectName()
		require.NoError(t, suite.ioctx.WriteFull(oid, []byte("x")))
		rdOp := CreateReadOp()
		defer rdOp.Release()
		es := rdOp.Exec("hello", "say_hello", nil)
		require.NoError(t, rdOp.Operate(suite.ioctx, oid, OperationNoFlag))

		out, err := es.Output()
		require.NoError(t, err)
		ta.Equal([]byte("Hello, world!"), out)
		bytes, err := es.Bytes()
		require.NoError(t, err)
		ta.Equal(out, bytes)
	})
}
