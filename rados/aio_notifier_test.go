//go:build ceph_preview

package rados

import (
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forEachAioMode runs f as a subtest once per AioMode, restoring
// AioModeCallback afterwards.
func (suite *RadosTestSuite) forEachAioMode(f func()) {
	modes := []struct {
		name string
		mode AioMode
	}{
		{"callback", AioModeCallback},
		{"pipe", AioModePipe},
	}
	for _, m := range modes {
		suite.Run(m.name, func() {
			SetAioMode(m.mode)
			defer SetAioMode(AioModeCallback)
			f()
		})
	}
}

func (suite *RadosTestSuite) TestAioNotifierPipeStops() {
	suite.SetupConnection()
	ta := assert.New(suite.T())
	defer SetAioMode(AioModeCallback)

	write := func(i int) *AioCompletion {
		wop := CreateWriteOp()
		defer wop.Release()
		wop.WriteFull([]byte("data"))
		c, err := wop.OperateAsync(suite.ioctx, fmt.Sprintf("%s_%d", suite.T().Name(), i), OperationNoFlag)
		require.NoError(suite.T(), err)
		return c
	}

	// warm up librados' callback threads before taking the baseline
	c := write(-1)
	<-c.Done()
	c.Release()
	baseline := goroutineCount()

	SetAioMode(AioModePipe)
	ta.Equal(baseline, goroutineCount(), "the drain goroutine starts lazily")
	completions := make([]*AioCompletion, 64)
	for i := range completions {
		completions[i] = write(i)
	}
	ta.Equal(baseline+1, goroutineCount(), "one drain goroutine")

	// switching modes waits for every in-flight pipe completion and stops
	// the drain goroutine
	SetAioMode(AioModeCallback)
	for i, c := range completions {
		select {
		case <-c.Done():
		default:
			ta.Failf("not done", "completion %d is still pending", i)
		}
		ta.Equal(0, c.ReturnValue())
		c.Release()
	}
	ta.Equal(baseline, goroutineCount())

	// the pipe can be set up again
	SetAioMode(AioModePipe)
	c = write(-2)
	<-c.Done()
	ta.NoError(c.Err())
	c.Release()
}

func (suite *RadosTestSuite) TestAioNotifierInvalidMode() {
	suite.SetupConnection()
	defer SetAioMode(AioModeCallback)

	SetAioMode(AioMode(-1))
	wop := CreateWriteOp()
	defer wop.Release()
	wop.WriteFull([]byte("data"))
	_, err := wop.OperateAsync(suite.ioctx, suite.GenObjectName(), OperationNoFlag)
	assert.Error(suite.T(), err)
	// the op keeps its steps and can still be operated
	SetAioMode(AioModeCallback)
	c, err := wop.OperateAsync(suite.ioctx, suite.GenObjectName(), OperationNoFlag)
	require.NoError(suite.T(), err)
	<-c.Done()
	assert.NoError(suite.T(), c.Err())
	c.Release()
}
