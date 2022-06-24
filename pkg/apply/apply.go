package apply

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultNamespace = "default"
)

type Apply interface {
	Apply(ctx context.Context, owner kclient.Object, objs ...kclient.Object) error
	WithOwnerSubContext(ownerSubContext string) Apply
	WithNamespace(ns string) Apply
	WithPruneGVKs(gvks ...schema.GroupVersionKind) Apply

	FindOwner(obj kclient.Object) (kclient.Object, error)
	PurgeOrphan(obj kclient.Object) error
}

func New(c kclient.Client) Apply {
	return &apply{
		client:           c,
		reconcilers:      defaultReconcilers,
		defaultNamespace: defaultNamespace,
	}
}
