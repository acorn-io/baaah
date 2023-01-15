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
	Ensure(ctx context.Context, obj ...kclient.Object) error
	Apply(ctx context.Context, owner kclient.Object, objs ...kclient.Object) error
	WithOwnerSubContext(ownerSubContext string) Apply
	WithNamespace(ns string) Apply
	WithPruneGVKs(gvks ...schema.GroupVersionKind) Apply
	WithNoPrune() Apply

	FindOwner(ctx context.Context, obj kclient.Object) (kclient.Object, error)
	PurgeOrphan(ctx context.Context, obj kclient.Object) error
}

func Ensure(ctx context.Context, client kclient.Client, obj ...kclient.Object) error {
	return New(client).Ensure(ctx, obj...)
}

func New(c kclient.Client) Apply {
	return &apply{
		client:           c,
		reconcilers:      defaultReconcilers,
		defaultNamespace: defaultNamespace,
	}
}
