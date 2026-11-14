//go:build ceph_preview

package rados

/*
#cgo LDFLAGS: -lrados
#include <stdlib.h>
#include <rados/librados.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unsafe"
)

// errEmptyKey is returned for a service map dict holding an empty key, which
// librados would read as the end of the dict.
var errEmptyKey = errors.New("key must not be empty")

// ServiceRegister registers the connection in the manager's service map as the
// named daemon of the named service, with metadata as the daemon's static
// metadata. librados adds metadata of its own, such as ceph_version and
// hostname, and a key it sets itself keeps librados's value. It keeps the
// registration for the lifetime of the connection: Shutdown removes it. A
// registration made before Connect is sent when the connection connects.
//
// A connection registers once. librados answers EBUSY while Connect runs, and
// after a Connect that failed in its monitor bootstrap, but keeps the
// registration all the same, so a retry is answered EEXIST.
//
// The service may not be a Ceph daemon type (osd, mds, client, mon or mgr).
// No metadata key may be empty, and no key or value may hold a NUL byte.
//
// Implements:
//
//	int rados_service_register(rados_t cluster,
//	                           const char *service,
//	                           const char *daemon,
//	                           const char *metadata_dict);
func (c *Conn) ServiceRegister(service, daemon string, metadata map[string]string) error {
	dict, err := encodeServiceDict(metadata)
	if err != nil {
		return fmt.Errorf("metadata: %w", err)
	}
	if c.cluster == nil {
		return ErrNotConnected
	}

	cService := C.CString(service)
	defer C.free(unsafe.Pointer(cService))
	cDaemon := C.CString(daemon)
	defer C.free(unsafe.Pointer(cDaemon))
	cDict := C.CBytes(dict)
	defer C.free(cDict)

	ret := C.rados_service_register(c.cluster, cService, cDaemon, (*C.char)(cDict))
	return getError(ret)
}

// ServiceUpdateStatus replaces the status the connection reports in the
// manager's service map, which the manager shows in its "service status"
// command. The connection must be connected, and the manager keeps the status
// only of a connection registered with ServiceRegister. librados sends it with
// its next report to the manager.
//
// No status key may be empty, and no key or value may hold a NUL byte.
//
// Implements:
//
//	int rados_service_update_status(rados_t cluster,
//	                                const char *status_dict);
func (c *Conn) ServiceUpdateStatus(status map[string]string) error {
	dict, err := encodeServiceDict(status)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}
	if c.cluster == nil {
		return ErrNotConnected
	}

	cDict := C.CBytes(dict)
	defer C.free(cDict)

	ret := C.rados_service_update_status(c.cluster, (*C.char)(cDict))
	return getError(ret)
}

// encodeServiceDict encodes m as librados's service map dict: each pair as
// its key, a NUL, its value and a NUL, then a closing NUL. The keys are
// sorted, so a map always has the same encoding.
func encodeServiceDict(m map[string]string) ([]byte, error) {
	keys := slices.Sorted(maps.Keys(m))
	size := 1
	for _, k := range keys {
		switch {
		case k == "":
			return nil, errEmptyKey
		case strings.IndexByte(k, 0) >= 0:
			return nil, fmt.Errorf("key %q: %w", k, ErrNulInString)
		case strings.IndexByte(m[k], 0) >= 0:
			return nil, fmt.Errorf("value of key %q: %w", k, ErrNulInString)
		}
		size += len(k) + len(m[k]) + 2
	}

	dict := make([]byte, 0, size)
	for _, k := range keys {
		dict = append(dict, k...)
		dict = append(dict, 0)
		dict = append(dict, m[k]...)
		dict = append(dict, 0)
	}
	return append(dict, 0), nil
}
