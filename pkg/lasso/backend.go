package lasso

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/acorn-io/baaah/pkg/backend"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/rancher/lasso/pkg/controller"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kcache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Backend struct {
	client.Client

	cacheFactory controller.SharedControllerFactory
	cache        cache.Cache
	started      bool
}

func NewBackend(cacheFactory controller.SharedControllerFactory, client client.Client, cache cache.Cache) *Backend {
	return &Backend{
		Client:       client,
		cacheFactory: cacheFactory,
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

func (b *Backend) Trigger(gvk schema.GroupVersionKind, key string, delay time.Duration) error {
	controller, err := b.cacheFactory.ForKind(gvk)
	if err != nil {
		return err
	}
	if delay > 0 {
		ns, name, ok := strings.Cut(key, "/")
		if ok {
			controller.EnqueueAfter(ns, name, delay)
		} else {
			controller.EnqueueAfter("", key, delay)
		}
	} else {
		controller.EnqueueKey(router.TriggerPrefix + key)
	}
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

func (b *Backend) GVKForObject(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error) {
	return apiutil.GVKForObject(obj, scheme)
}

func (b *Backend) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (kcache.SharedIndexInformer, error) {
	return b.cacheFactory.SharedCacheFactory().ForKind(gvk)
}
