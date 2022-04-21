package backend

import (
	"context"

	"github.com/acorn-io/baaah/pkg/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type Callback func(gvk schema.GroupVersionKind, key string, obj runtime.Object) (runtime.Object, error)

type Trigger interface {
	Trigger(gvk schema.GroupVersionKind, key string) error
}

type Watcher interface {
	Watch(ctx context.Context, gvk schema.GroupVersionKind, name string, cb Callback) error
}

type Backend interface {
	Trigger
	Watcher
	Reader
	Writer
	CacheFactory

	Start(ctx context.Context) error
}

type CacheFactory interface {
	GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (cache.SharedIndexInformer, error)
}

type Writer interface {
	Delete(ctx context.Context, obj meta.Object) error
	Update(ctx context.Context, obj meta.Object) error
	UpdateStatus(ctx context.Context, obj meta.Object) error
	Create(ctx context.Context, obj meta.Object) error
}

type Reader interface {
	Get(ctx context.Context, obj meta.Object, name string, opts *meta.GetOptions) error
	List(ctx context.Context, obj meta.ObjectList, opts *meta.ListOptions) error
	GVKForObject(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error)
}
