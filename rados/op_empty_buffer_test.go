package rados

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each case used to panic with an index out of range on the empty slice.

func (suite *RadosTestSuite) TestWriteOpSetXattrEmpty() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	for _, value := range [][]byte{nil, {}} {
		oid := suite.GenObjectName()
		op := CreateWriteOp()
		op.Create(CreateIdempotent)
		op.SetXattr("empty", value)
		ta.NoError(op.Operate(suite.ioctx, oid, OperationNoFlag))
		op.Release()

		buf := make([]byte, 16)
		n, err := suite.ioctx.GetXattr(oid, "empty", buf)
		ta.NoError(err)
		ta.Equal(0, n)
	}
}

func (suite *RadosTestSuite) TestWriteOpWriteFullEmpty() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	for _, data := range [][]byte{nil, {}} {
		oid := suite.GenObjectName()
		op := CreateWriteOp()
		op.WriteFull(data)
		ta.NoError(op.Operate(suite.ioctx, oid, OperationNoFlag))
		op.Release()

		stat, err := suite.ioctx.Stat(oid)
		ta.NoError(err)
		ta.Equal(uint64(0), stat.Size)
	}

	// an empty WriteFull truncates an existing object
	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.Write(oid, []byte("data"), 0))
	op := CreateWriteOp()
	defer op.Release()
	op.WriteFull(nil)
	ta.NoError(op.Operate(suite.ioctx, oid, OperationNoFlag))
	stat, err := suite.ioctx.Stat(oid)
	ta.NoError(err)
	ta.Equal(uint64(0), stat.Size)
}

func (suite *RadosTestSuite) TestWriteOpWriteEmpty() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.Write(oid, []byte("data"), 0))
	op := CreateWriteOp()
	defer op.Release()
	op.Write([]byte{}, 0)
	ta.NoError(op.Operate(suite.ioctx, oid, OperationNoFlag))

	buf := make([]byte, 16)
	n, err := suite.ioctx.Read(oid, buf, 0)
	ta.NoError(err)
	ta.Equal("data", string(buf[:n]))
}

func (suite *RadosTestSuite) TestWriteOpWriteSameEmpty() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.Write(oid, []byte("data"), 0))

	// the OSD accepts a zero-length writesame as a no-op
	op := CreateWriteOp()
	op.WriteSame([]byte{}, 0, 0)
	ta.NoError(op.Operate(suite.ioctx, oid, OperationNoFlag))
	op.Release()

	// but rejects an empty pattern with a nonzero length
	op = CreateWriteOp()
	op.WriteSame(nil, 8, 0)
	ta.Error(op.Operate(suite.ioctx, oid, OperationNoFlag))
	op.Release()

	buf := make([]byte, 16)
	n, err := suite.ioctx.Read(oid, buf, 0)
	ta.NoError(err)
	ta.Equal("data", string(buf[:n]))
}

func (suite *RadosTestSuite) TestReadOpReadEmpty() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.Write(oid, []byte("data"), 0))

	op := CreateReadOp()
	defer op.Release()
	step := op.Read(0, nil)
	ta.NoError(op.Operate(suite.ioctx, oid, OperationNoFlag))
	ta.Equal(int64(0), step.BytesRead)
}

func (suite *RadosTestSuite) TestWriteOpSetOmapEmpty() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	op := CreateWriteOp()
	op.Create(CreateIdempotent)
	op.SetOmap(map[string][]byte{"nil": nil, "empty": {}})
	ta.NoError(op.Operate(suite.ioctx, oid, OperationNoFlag))
	op.Release()

	// an empty map is a no-op
	op = CreateWriteOp()
	op.SetOmap(map[string][]byte{})
	ta.NoError(op.Operate(suite.ioctx, oid, OperationNoFlag))
	op.Release()

	vals, err := suite.ioctx.GetOmapValues(oid, "", "", 10)
	require.NoError(suite.T(), err)
	ta.Len(vals, 2)
	ta.Empty(vals["nil"])
	ta.Empty(vals["empty"])
}
