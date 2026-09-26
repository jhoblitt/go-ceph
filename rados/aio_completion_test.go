//go:build ceph_preview

package rados

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goroutineCount returns runtime.NumGoroutine less the goroutines that
// stand for C threads which have called back into Go. Since Go 1.21 such a
// thread keeps its goroutine, idle in a syscall, until the thread exits;
// their number is bounded by librados' thread pool, not by the number of
// operations.
func goroutineCount() int {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	idle := 0
	for _, g := range strings.Split(string(buf), "\n\n") {
		lines := strings.Split(strings.TrimSpace(g), "\n")
		if len(lines) == 3 &&
			strings.HasSuffix(lines[0], "[syscall, locked to thread]:") &&
			strings.HasPrefix(lines[1], "runtime.goexit(") {
			idle++
		}
	}
	return runtime.NumGoroutine() - idle
}

func (suite *RadosTestSuite) TestAioCompletionConcurrent() {
	suite.SetupConnection()
	ta := assert.New(suite.T())

	const inflight = 256
	write := func(i int) *AioCompletion {
		wop := CreateWriteOp()
		defer wop.Release()
		wop.WriteFull([]byte(fmt.Sprintf("object %d", i)))
		c, err := wop.OperateAsync(suite.ioctx, fmt.Sprintf("%s_%d", suite.T().Name(), i), OperationNoFlag)
		require.NoError(suite.T(), err)
		return c
	}

	// a warm-up operation starts anything that lives for the process
	c := write(-1)
	<-c.Done()
	c.Release()
	baseline := goroutineCount()

	completions := make([]*AioCompletion, inflight)
	for i := range completions {
		completions[i] = write(i)
	}
	for i, c := range completions {
		select {
		case <-c.Done():
		case <-time.After(time.Minute):
			suite.T().Fatalf("completion %d never finished", i)
		}
		ta.Equal(0, c.ReturnValue())
		c.Release()
	}

	// polled here rather than with assert.Eventually, which runs its
	// condition in a goroutine of its own
	deadline := time.Now().Add(10 * time.Second)
	for goroutineCount() > baseline && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	ta.LessOrEqual(goroutineCount(), baseline)

	buf := make([]byte, 64)
	n, err := suite.ioctx.Read(fmt.Sprintf("%s_%d", suite.T().Name(), inflight-1), buf, 0)
	ta.NoError(err)
	ta.Equal(fmt.Sprintf("object %d", inflight-1), string(buf[:n]))
}
