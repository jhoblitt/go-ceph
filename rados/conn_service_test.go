//go:build ceph_preview

package rados

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// serviceMapTimeout bounds a wait for the manager, which publishes its
	// service map every mgr_tick_period (2s by default) and hears a status
	// with the client's next report, every mgr_stats_period (5s by default).
	serviceMapTimeout = time.Minute
	// serviceCloseTimeout is under mgr_service_beacon_grace (1 minute by
	// default), after which the manager drops a daemon whose beacons stopped,
	// so only the close Shutdown sends removes the entry within it.
	serviceCloseTimeout = 30 * time.Second
)

// errnoOf returns the error code a rados error carries, or 0.
func errnoOf(err error) int {
	var ec interface{ ErrorCode() int }
	if !errors.As(err, &ec) {
		return 0
	}
	return ec.ErrorCode()
}

// refusedServiceDicts are maps librados's dict encoding cannot carry: it reads
// an empty key as the end of the dict and ends every string at a NUL.
var refusedServiceDicts = []struct {
	dict map[string]string
	want error
}{
	{map[string]string{"a": "b", "": "c"}, errEmptyKey},
	{map[string]string{"a": "b", "c\x00d": "e"}, ErrNulInString},
	{map[string]string{"a": "b", "c": "d\x00e"}, ErrNulInString},
}

func TestEncodeServiceDict(t *testing.T) {
	for _, tc := range []struct {
		name string
		dict map[string]string
		want string
	}{
		{"nil", nil, "\x00"},
		{"empty", map[string]string{}, "\x00"},
		// librados.h's own example
		{"pairs", map[string]string{"this": "that", "foo": "bar"}, "foo\x00bar\x00this\x00that\x00\x00"},
		{"empty value", map[string]string{"a": "", "b": "c"}, "a\x00\x00b\x00c\x00\x00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := encodeServiceDict(tc.dict)
			require.NoError(t, err)
			assert.Equal(t, []byte(tc.want), got)
		})
	}
}

func TestEncodeServiceDictRefused(t *testing.T) {
	for _, r := range refusedServiceDicts {
		got, err := encodeServiceDict(r.dict)
		assert.ErrorIs(t, err, r.want, "%q", r.dict)
		assert.Nil(t, got)
	}
}

// librados checks a registration, and answers a status update, before it
// needs a cluster, so these run on handles that never connect.
func TestServiceRegisterUnconnected(t *testing.T) {
	newConn := func(t *testing.T) *Conn {
		conn, err := NewConn()
		require.NoError(t, err)
		t.Cleanup(conn.Shutdown)
		return conn
	}

	t.Run("refused", func(t *testing.T) {
		conn := newConn(t)
		for _, r := range refusedServiceDicts {
			err := conn.ServiceRegister("gocephtest", "a", r.dict)
			assert.ErrorIs(t, err, r.want, "%q", r.dict)
		}
		for _, names := range [][2]string{
			{"osd", "a"}, {"mds", "a"}, {"client", "a"}, {"mon", "a"}, {"mgr", "a"},
			{"", "a"}, {"gocephtest", ""},
		} {
			err := conn.ServiceRegister(names[0], names[1], nil)
			assert.Equal(t, -int(syscall.EINVAL), errnoOf(err), "%q", names)
		}
		// librados refuses a second registration, so none of the refused
		// ones registered
		require.NoError(t, conn.ServiceRegister("gocephtest", "a", map[string]string{"a": "b"}))
		err := conn.ServiceRegister("gocephtest", "b", nil)
		assert.Equal(t, -int(syscall.EEXIST), errnoOf(err))
	})

	t.Run("status", func(t *testing.T) {
		conn := newConn(t)
		err := conn.ServiceUpdateStatus(map[string]string{"a": "b"})
		assert.Equal(t, -int(syscall.ENOTCONN), errnoOf(err))
		// refused before librados, which would answer ENOTCONN
		for _, r := range refusedServiceDicts {
			err := conn.ServiceUpdateStatus(r.dict)
			assert.ErrorIs(t, err, r.want, "%q", r.dict)
		}
	})

	t.Run("after shutdown", func(t *testing.T) {
		conn, err := NewConn()
		require.NoError(t, err)
		conn.Shutdown()
		assert.ErrorIs(t, conn.ServiceRegister("gocephtest", "a", nil), ErrNotConnected)
		assert.ErrorIs(t, conn.ServiceUpdateStatus(nil), ErrNotConnected)
	})
}

// serviceMapDaemon is a daemon's entry in the manager's `service dump`.
type serviceMapDaemon struct {
	GID      uint64            `json:"gid"`
	Metadata map[string]string `json:"metadata"`
}

func (suite *RadosTestSuite) mgrJSON(prefix string, v any) error {
	cmd, err := json.Marshal(map[string]string{"prefix": prefix, "format": "json"})
	if err != nil {
		return err
	}
	buf, _, err := suite.conn.MgrCommand([][]byte{cmd})
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, v)
}

// serviceDump returns the daemons `service dump` lists under service.
func (suite *RadosTestSuite) serviceDump(service string) (map[string]serviceMapDaemon, error) {
	var dump struct {
		Services map[string]struct {
			Daemons map[string]json.RawMessage `json:"daemons"`
		} `json:"services"`
	}
	if err := suite.mgrJSON("service dump", &dump); err != nil {
		return nil, err
	}
	daemons := map[string]serviceMapDaemon{}
	for name, raw := range dump.Services[service].Daemons {
		// the service's summary string sits beside its daemons
		if name == "summary" {
			continue
		}
		var d serviceMapDaemon
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, err
		}
		daemons[name] = d
	}
	return daemons, nil
}

// serviceStatus returns the status `service status` lists for each daemon of
// service.
func (suite *RadosTestSuite) serviceStatus(service string) (map[string]map[string]string, error) {
	var status map[string]map[string]struct {
		Status map[string]string `json:"status"`
	}
	if err := suite.mgrJSON("service status", &status); err != nil {
		return nil, err
	}
	daemons := map[string]map[string]string{}
	for name, d := range status[service] {
		daemons[name] = d.Status
	}
	return daemons, nil
}

// waitFor polls cond in the calling goroutine, unlike assert.Eventually,
// until it holds or timeout passes, and reports whether it held.
func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(500 * time.Millisecond)
	}
	return true
}

// newServiceConn returns a connection for a test to register and shut down,
// leaving suite.conn connected for the manager commands that read the result.
func (suite *RadosTestSuite) newServiceConn() *Conn {
	conn, err := NewConn()
	require.NoError(suite.T(), err)
	require.NoError(suite.T(), conn.ReadDefaultConfigFile())
	require.NoError(suite.T(), conn.Connect())
	return conn
}

// newServiceName returns a service name unique to the run: the test
// cluster's gateways register under rgw.
func newServiceName() string {
	return fmt.Sprintf("gocephtest%d", time.Now().UnixNano())
}

func (suite *RadosTestSuite) TestServiceRegister() {
	suite.SetupConnection()
	ta := assert.New(suite.T())
	service := newServiceName()
	conn := suite.newServiceConn()
	defer conn.Shutdown()

	for _, r := range refusedServiceDicts {
		ta.ErrorIs(conn.ServiceRegister(service, "a", r.dict), r.want, "%q", r.dict)
	}
	ta.Equal(-int(syscall.EINVAL), errnoOf(conn.ServiceRegister("osd", "a", nil)))

	daemon := strconv.FormatUint(conn.GetInstanceID(), 10)
	metadata := map[string]string{"go_ceph_test": "service_register", "empty_value": ""}
	require.NoError(suite.T(), conn.ServiceRegister(service, daemon, metadata))

	var (
		daemons map[string]serviceMapDaemon
		err     error
	)
	listed := waitFor(serviceMapTimeout, func() bool {
		daemons, err = suite.serviceDump(service)
		return err == nil && len(daemons) > 0
	})
	require.True(suite.T(), listed, "%s never in service dump: %v", service, err)
	// none of the refused registrations reached the map
	require.Len(suite.T(), daemons, 1)
	require.Contains(suite.T(), daemons, daemon)
	got := daemons[daemon]
	ta.Equal(conn.GetInstanceID(), got.GID)
	for k, v := range metadata {
		if ta.Contains(got.Metadata, k) {
			ta.Equal(v, got.Metadata[k], k)
		}
	}
	ta.NotEmpty(got.Metadata["ceph_version"])
	ta.NotEmpty(got.Metadata["hostname"])

	conn.Shutdown()
	removed := waitFor(serviceCloseTimeout, func() bool {
		daemons, err = suite.serviceDump(service)
		return err == nil && len(daemons) == 0
	})
	ta.True(removed, "%s still in service dump after Shutdown: %v %v", service, daemons, err)
}

func (suite *RadosTestSuite) TestServiceUpdateStatus() {
	suite.SetupConnection()
	service := newServiceName()
	conn := suite.newServiceConn()
	defer conn.Shutdown()

	daemon := strconv.FormatUint(conn.GetInstanceID(), 10)
	require.NoError(suite.T(), conn.ServiceRegister(service, daemon, nil))
	status := map[string]string{"go_ceph_test": "service_update_status", "empty_value": ""}
	require.NoError(suite.T(), conn.ServiceUpdateStatus(status))

	var (
		daemons map[string]map[string]string
		err     error
	)
	// the status is not part of the service map, so only `service status`
	// shows it
	shown := waitFor(serviceMapTimeout, func() bool {
		daemons, err = suite.serviceStatus(service)
		return err == nil && assert.ObjectsAreEqual(status, daemons[daemon])
	})
	assert.True(suite.T(), shown, "%s.%s status never shown: %v %v", service, daemon, daemons, err)
}
