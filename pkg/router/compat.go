package router

import (
	"context"

	"github.com/acorn-io/baaah/pkg/meta"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
)

func ToReader(getter Getter) client2.Reader {
	return readerWrapper{getter}
}

func FromReader(ctx context.Context, c client2.Reader) Reader {
	return &getterWrapper{
		ctx: ctx,
		c:   c,
	}
}

type getterWrapper struct {
	c   client2.Reader
	ctx context.Context
}

func (g *getterWrapper) Get(obj meta.Object, name string, opts *meta.GetOptions) error {
	return g.c.Get(g.ctx, client2.ObjectKey{
		Name:      name,
		Namespace: opts.GetNamespace(""),
	}, obj)
}

func (g *getterWrapper) List(obj meta.ObjectList, opts *meta.ListOptions) error {
	return g.c.List(g.ctx, obj, &client2.ListOptions{
		Namespace: opts.GetNamespace(""),
	})
}

type readerWrapper struct {
	getter Getter
}

func (r readerWrapper) Get(ctx context.Context, key client2.ObjectKey, obj client2.Object) error {
	return r.getter.Get(obj, key.Name, &meta.GetOptions{
		Namespace: key.Namespace,
	})
}

func (r readerWrapper) List(ctx context.Context, list client2.ObjectList, opts ...client2.ListOption) error {
	panic("implement me")
}
