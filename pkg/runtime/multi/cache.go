package multi

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewCacheNotFoundError(group string) error {
	return &CacheNotFoundError{group: group}
}

type CacheNotFoundError struct {
	group string
}

func (c *CacheNotFoundError) Error() string {
	return fmt.Sprintf("cache for group %s not found", c.group)
}

func NewCache(defaultCache cache.Cache, defaultScheme *runtime.Scheme, caches map[string]cache.Cache, schemes map[string]*runtime.Scheme) cache.Cache {
	newScheme := runtime.NewScheme()
	gvksSeen := make(map[schema.GroupVersionKind]struct{})
	groups := make(map[string]struct{})
	for group := range schemes {
		groups[group] = struct{}{}
	}

	for group, scheme := range schemes {
		_, inGroups := groups[group]
		for key, val := range scheme.AllKnownTypes() {
			if _, ok := gvksSeen[key]; !ok && inGroups && key.Group == group {
				newScheme.AddKnownTypeWithName(key, reflect.New(val).Interface().(runtime.Object))
				gvksSeen[key] = struct{}{}
			}
		}
	}

	for key, val := range defaultScheme.AllKnownTypes() {
		if _, ok := gvksSeen[key]; !ok {
			newScheme.AddKnownTypeWithName(key, reflect.New(val).Interface().(runtime.Object))
			gvksSeen[key] = struct{}{}
		}
	}

	return multiCache{
		defaultCache: defaultCache,
		caches:       caches,
		scheme:       newScheme,
	}
}

type multiCache struct {
	defaultCache cache.Cache
	caches       map[string]cache.Cache
	scheme       *runtime.Scheme
}

func (m multiCache) Get(ctx context.Context, key kclient.ObjectKey, obj kclient.Object, opts ...kclient.GetOption) error {
	c, err := m.getCache(obj)
	if err != nil {
		return err
	}
	return c.Get(ctx, key, obj, opts...)
}

func (m multiCache) List(ctx context.Context, list kclient.ObjectList, opts ...kclient.ListOption) error {
	c, err := m.getCache(list)
	if err != nil {
		return err
	}
	return c.List(ctx, list, opts...)
}

func (m multiCache) GetInformer(ctx context.Context, obj kclient.Object) (cache.Informer, error) {
	c, err := m.getCache(obj)
	if err != nil {
		return nil, err
	}
	return c.GetInformer(ctx, obj)
}

func (m multiCache) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (cache.Informer, error) {
	c, err := m.getCacheForGroup(gvk.Group)
	if err != nil {
		return nil, err
	}
	return c.GetInformerForKind(ctx, gvk)
}

func (m multiCache) Start(ctx context.Context) error {
	go func() {
		if err := m.defaultCache.Start(ctx); err != nil {
			panic(err)
		}
	}()

	for _, c := range m.caches {
		go func(c cache.Cache) {
			if err := c.Start(ctx); err != nil {
				panic(err)
			}
		}(c)
	}

	return nil
}

func (m multiCache) WaitForCacheSync(ctx context.Context) bool {
	if !m.defaultCache.WaitForCacheSync(ctx) {
		return false
	}

	for _, c := range m.caches {
		if !c.WaitForCacheSync(ctx) {
			return false
		}
	}
	return true
}

func (m multiCache) IndexField(ctx context.Context, obj kclient.Object, field string, extractValue kclient.IndexerFunc) error {
	c, err := m.getCache(obj)
	if err != nil {
		return err
	}
	return c.IndexField(ctx, obj, field, extractValue)
}

func (m multiCache) getCache(obj runtime.Object) (cache.Cache, error) {
	gvk, err := apiutil.GVKForObject(obj, m.scheme)
	if err != nil {
		return nil, err
	}
	return m.getCacheForGroup(gvk.Group)
}

func (m multiCache) getCacheForGroup(group string) (cache.Cache, error) {
	if c, ok := m.caches[group]; ok {
		return c, nil
	} else if m.defaultCache != nil {
		return m.defaultCache, nil
	}

	return nil, NewCacheNotFoundError(group)
}
