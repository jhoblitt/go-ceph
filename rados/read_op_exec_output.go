//go:build ceph_preview

package rados

// #include <stdlib.h>
import "C"

import (
	"unsafe"
)

// Output returns the output of the executed class method whatever the
// method returned. A method may fill its output and then fail, and librados
// hands that output over too, while Bytes returns an error for a method that
// returned a negative value.
func (es *ReadOpExecStep) Output() ([]byte, error) {
	if !es.completed {
		return nil, ErrOperationIncomplete
	}

	return C.GoBytes(unsafe.Pointer(es.outBuffPtr), C.int(es.outBuffLen)), nil
}
