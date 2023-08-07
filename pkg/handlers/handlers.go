package handlers

import (
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
)

// GCOrphans will delete an object whose owner has been deleted.
func GCOrphans(req router.Request, _ router.Response) error {
	return apply.New(req.Client).PurgeOrphan(req.Ctx, req.Object)
}

// DoNothing is a handler that does nothing. It is useful because resp.Objects will only work if a GVK is being watched.
// This ensures that resp.Objects does what it is supposed to for objects that are no longer passed to resp.Objects, and
// are not watched anywhere else.
func DoNothing(router.Request, router.Response) error {
	return nil
}
