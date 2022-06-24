package router

import (
	"context"
	"fmt"
	"reflect"

	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/backend"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kcache "k8s.io/client-go/tools/cache"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type save struct {
	apply  apply.Apply
	cache  backend.CacheFactory
	client kclient.Client
}

func (s *save) save(unmodified runtime.Object, req Request, resp *response, watchingGVKS []schema.GroupVersionKind) (kclient.Object, error) {
	var owner = req.Object
	if owner == nil {
		owner := &unstructured.Unstructured{}
		owner.SetGroupVersionKind(req.GVK)
		owner.SetNamespace(req.Namespace)
		owner.SetName(req.Name)
	}
	apply := s.apply.
		WithPruneGVKs(watchingGVKS...)

	if err := apply.Apply(req.Ctx, owner, resp.objects...); err != nil {
		return nil, err
	}

	newObj := req.Object
	if newObj != nil && statusChanged(unmodified, newObj) {
		return newObj, s.client.Status().Update(req.Ctx, newObj)
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
