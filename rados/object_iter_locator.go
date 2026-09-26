//go:build ceph_preview

package rados

// Locator returns the object locator of the current value of the iterator,
// after a successful call to Next. It is empty for an object stored
// without a locator.
//
// Implements:
//
//	int rados_nobjects_list_next(rados_list_ctx_t ctx,
//	                             const char **entry,
//	                             const char **key,
//	                             const char **nspace);
func (iter *Iter) Locator() string {
	if iter.err != nil {
		return ""
	}
	return iter.locator
}
