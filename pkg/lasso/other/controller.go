package other

import (
	"context"
	"time"

	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/generic"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Handler[T meta.Object] func(string, T) (T, error)

type Controller[T any, TP PObject[T]] interface {
	OnChange(ctx context.Context, name string, sync Handler[TP])
	OnRemove(ctx context.Context, name string, sync Handler[TP])
	Enqueue(namespace, name string)
	EnqueueAfter(namespace, name string, duration time.Duration)
	Client() Client[T, TP]
	Cache() Cache[T, TP]
}

type objController[T any, TP PObject[T]] struct {
	controller    controller.SharedController
	client        *client.Client
	gvk           schema.GroupVersionKind
	groupResource schema.GroupResource
}

func NewController[T any, TP PObject[T]](gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) Controller[T, TP] {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &objController[T, TP]{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func (c *objController[T, TP]) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Client().Update(obj.(TP))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func (c *objController[T, TP]) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *objController[T, TP]) OnChange(ctx context.Context, name string, sync Handler[TP]) {
	c.AddGenericHandler(ctx, name, func(key string, obj runtime.Object) (runtime.Object, error) {
		return sync(key, obj.(TP))
	})
}

func (c *objController[T, TP]) OnRemove(ctx context.Context, name string, sync Handler[TP]) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), func(key string, obj runtime.Object) (runtime.Object, error) {
		return sync(key, obj.(TP))
	}))
}

func (c *objController[T, TP]) Enqueue(namespace, name string) {
	c.controller.Enqueue(namespace, name)
}

func (c *objController[T, TP]) EnqueueAfter(namespace, name string, duration time.Duration) {
	c.controller.EnqueueAfter(namespace, name, duration)
}

func (c *objController[T, TP]) Client() Client[T, TP] {
	return &objClient[T, TP]{
		client: c.client,
	}
}

func (c *objController[T, TP]) Cache() Cache[T, TP] {
	return &objCache[T, TP]{
		indexer:  c.controller.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}
