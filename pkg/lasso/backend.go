package lasso

import (
	"context"
	"fmt"

	"github.com/ibuildthecloud/baaah/pkg/backend"
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/rancher/lasso/pkg/controller"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kcache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Backend struct {
	cacheFactory controller.SharedControllerFactory
	cache        cache.Cache
	client       client.Client
	started      bool
}

func NewBackend(cacheFactory controller.SharedControllerFactory, client client.Client, cache cache.Cache) *Backend {
	return &Backend{
		cacheFactory: cacheFactory,
		client:       client,
		cache:        cache,
	}
}

func (b *Backend) Start(ctx context.Context) error {
	if err := b.cacheFactory.Start(ctx, 5); err != nil {
		return err
	}
	go func() {
		if err := b.cache.Start(ctx); err != nil {
			panic(err)
		}
	}()
	if !b.cache.WaitForCacheSync(ctx) {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	return nil
}

func (b *Backend) Trigger(gvk schema.GroupVersionKind, key string) error {
	controller, err := b.cacheFactory.ForKind(gvk)
	if err != nil {
		return err
	}
	controller.EnqueueKey(key)
	return nil
}

func (b *Backend) Watch(ctx context.Context, gvk schema.GroupVersionKind, name string, cb backend.Callback) error {
	c, err := b.cacheFactory.ForKind(gvk)
	if err != nil {
		return err
	}
	handler := controller.SharedControllerHandlerFunc(func(key string, obj runtime.Object) (runtime.Object, error) {
		return cb(gvk, key, obj)
	})
	c.RegisterHandler(ctx, fmt.Sprintf("%s %v", name, gvk), handler)
	return c.Start(ctx, 5)
}

func (b *Backend) Get(ctx context.Context, obj meta.Object, name string, opts *meta.GetOptions) error {
	return b.client.Get(ctx, client.ObjectKey{
		Namespace: opts.GetNamespace(""),
		Name:      name,
	}, obj)
}

func (b *Backend) List(ctx context.Context, obj meta.ObjectList, opts *meta.ListOptions) error {
	return b.client.List(ctx, obj, &client.ListOptions{
		Namespace:     opts.GetNamespace(""),
		LabelSelector: opts.GetSelector(),
	})
}

func (b *Backend) GVKForObject(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error) {
	return apiutil.GVKForObject(obj, scheme)
}

func (b *Backend) Delete(ctx context.Context, obj meta.Object) error {
	return b.client.Delete(ctx, obj)
}

func (b *Backend) Update(ctx context.Context, obj meta.Object) error {
	return b.client.Update(ctx, obj)
}

func (b *Backend) UpdateStatus(ctx context.Context, obj meta.Object) error {
	return b.client.Status().Update(ctx, obj)
}

func (b *Backend) Create(ctx context.Context, obj meta.Object) error {
	return b.client.Create(ctx, obj)
}

func (b *Backend) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (kcache.SharedIndexInformer, error) {
	return b.cacheFactory.SharedCacheFactory().ForKind(gvk)
}
