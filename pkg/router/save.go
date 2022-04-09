package router

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ibuildthecloud/baaah/pkg/backend"
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/rancher/wrangler/pkg/apply"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kcache "k8s.io/client-go/tools/cache"
)

type save struct {
	setID  string
	apply  apply.Apply
	cache  backend.CacheFactory
	client backend.Writer
}

func (s *save) save(unmodified runtime.Object, req Request, resp *response, watchingGVKS []schema.GroupVersionKind) (meta.Object, error) {
	apply := s.apply.
		WithContext(req.Ctx).
		WithSetID(s.setID).
		WithCacheTypeFactory(saveInformerFactory{
			ctx:   req.Ctx,
			cache: s.cache,
		}).
		WithSetOwnerReference(true, false).
		WithOwnerKey(req.Key, req.GVK).
		WithOwner(unmodified).
		WithGVK(watchingGVKS...)

	objs := make([]runtime.Object, 0, len(resp.objects))
	for _, obj := range resp.objects {
		objs = append(objs, obj)
	}

	if err := apply.ApplyObjects(objs...); err != nil {
		return nil, err
	}

	newObj := req.Object
	if newObj != nil && statusChanged(unmodified, newObj) {
		return newObj, s.client.UpdateStatus(req.Ctx, newObj)
	}

	return newObj, nil
}

func statusField(obj runtime.Object) interface{} {
	v := reflect.ValueOf(obj).Elem()
	return v.FieldByName("Status").Interface()
}

func statusChanged(unmodified, newObj runtime.Object) bool {
	return !equality.Semantic.DeepEqual(statusField(unmodified), statusField(newObj))
}

type saveInformerFactory struct {
	ctx   context.Context
	cache backend.CacheFactory
}

func (s saveInformerFactory) Get(gvk schema.GroupVersionKind, gvr schema.GroupVersionResource) (kcache.SharedIndexInformer, error) {
	inf, err := s.cache.GetInformerForKind(s.ctx, gvk)
	if err != nil {
		return nil, err
	}
	if si, ok := inf.(kcache.SharedIndexInformer); ok {
		return si, nil
	}
	return nil, fmt.Errorf("informer of type %T is not a cache.SharedIndexInformer", inf)
}
