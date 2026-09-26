//go:build ceph_preview

package rados

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *RadosTestSuite) TestObjectIteratorLocator() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	withLocator := suite.GenObjectName()
	withoutLocator := suite.GenObjectName()
	suite.ioctx.SetLocator("the-locator")
	ta.NoError(suite.ioctx.WriteFull(withLocator, []byte("located")))
	suite.ioctx.SetLocator("")
	ta.NoError(suite.ioctx.WriteFull(withoutLocator, []byte("plain")))

	iter, err := suite.ioctx.Iter()
	require.NoError(suite.T(), err)
	defer iter.Close()
	locators := map[string]string{}
	for iter.Next() {
		locators[iter.Value()] = iter.Locator()
	}
	ta.NoError(iter.Err())

	ta.Contains(locators, withLocator)
	ta.Equal("the-locator", locators[withLocator])
	ta.Contains(locators, withoutLocator)
	ta.Equal("", locators[withoutLocator])
}
