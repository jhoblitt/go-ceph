//go:build ceph_preview

package rados

import (
	"github.com/stretchr/testify/assert"
)

func (suite *RadosTestSuite) TestReadOpGetOmapKeys() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	oid := suite.GenObjectName()
	ta.NoError(suite.ioctx.SetOmap(oid, map[string][]byte{
		"a": []byte("1"),
		"b": []byte("2"),
		"c": []byte("3"),
	}))

	getKeys := func(startAfter string, maxReturn uint64) ([]string, bool, error) {
		rop := CreateReadOp()
		defer rop.Release()
		step := rop.GetOmapKeys(startAfter, maxReturn)
		if _, err := step.Keys(); err != ErrOperationIncomplete {
			return nil, false, err
		}
		if err := rop.Operate(suite.ioctx, oid, OperationNoFlag); err != nil {
			return nil, false, err
		}
		keys, err := step.Keys()
		return keys, step.More(), err
	}

	keys, more, err := getKeys("", 2)
	ta.NoError(err)
	ta.Equal([]string{"a", "b"}, keys)
	ta.True(more)

	keys, more, err = getKeys("b", 10)
	ta.NoError(err)
	ta.Equal([]string{"c"}, keys)
	ta.False(more)

	keys, more, err = getKeys("c", 10)
	ta.NoError(err)
	ta.Empty(keys)
	ta.False(more)

	_, _, err = getKeys("a\x00b", 10)
	ta.ErrorIs(err, ErrNulInString)
}

func (suite *RadosTestSuite) TestReadOpGetOmapKeysAsync() {
	suite.SetupConnection()
	suite.forEachAioMode(func() {
		ta := assert.New(suite.T())

		oid := suite.GenObjectName()
		ta.NoError(suite.ioctx.SetOmap(oid, map[string][]byte{
			"a": []byte("1"),
			"b": []byte("2"),
			"c": []byte("3"),
		}))

		rop := CreateReadOp()
		step := rop.GetOmapKeys("a", 1)
		c, err := rop.OperateAsync(suite.ioctx, oid, OperationNoFlag)
		ta.NoError(err)
		rop.Release()
		<-c.Done()
		ta.NoError(c.Err())
		c.Release()
		keys, err := step.Keys()
		ta.NoError(err)
		ta.Equal([]string{"b"}, keys)
		ta.True(step.More())
	})
}
