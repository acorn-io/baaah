package router

import (
	"context"

	"github.com/ibuildthecloud/baaah/pkg/meta"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
)

func ToReader(getter Getter) client2.Reader {
	return readerWrapper{getter}
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
