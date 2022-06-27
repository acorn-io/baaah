package apply

import (
	"context"

	"github.com/acorn-io/baaah/pkg/apply/objectset"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// reconciler return false if it did not handle this object
type reconciler func(oldObj kclient.Object, newObj kclient.Object) (bool, error)

type apply struct {
	ctx              context.Context
	client           kclient.Client
	defaultNamespace string
	listerNamespace  string
	pruneTypes       map[schema.GroupVersionKind]bool
	reconcilers      map[schema.GroupVersionKind]reconciler
	ownerSubContext  string
	owner            kclient.Object
}

func (a apply) Apply(ctx context.Context, owner kclient.Object, objs ...kclient.Object) error {
	a.ctx = ctx
	a.owner = owner
	os, err := objectset.NewObjectSet(a.client.Scheme(), objs...)
	if err != nil {
		return err
	}
	return a.apply(os)
}

// WithPruneGVKs uses a known listing of existing gvks to modify the the prune types to allow for deletion of objects
func (a apply) WithPruneGVKs(gvks ...schema.GroupVersionKind) Apply {
	pruneTypes := make(map[schema.GroupVersionKind]bool, len(gvks))
	for k, v := range a.pruneTypes {
		pruneTypes[k] = v
	}
	for _, gvk := range gvks {
		pruneTypes[gvk] = true
	}
	a.pruneTypes = pruneTypes
	return a
}

func (a apply) WithNamespace(ns string) Apply {
	a.listerNamespace = ns
	a.defaultNamespace = ns
	return a
}

func (a apply) WithOwnerSubContext(ownerSubContext string) Apply {
	a.ownerSubContext = ownerSubContext
	return a
}
