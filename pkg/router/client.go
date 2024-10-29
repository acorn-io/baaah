package router

import (
	"context"

	"github.com/acorn-io/baaah/pkg/backend"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type TriggerRegistry interface {
	Watch(obj runtime.Object, namespace, name string, selector labels.Selector, fields fields.Selector) error
	WatchingGVKs() []schema.GroupVersionKind
}

type client struct {
	backend backend.Backend
	reader
	writer
	status
}

func (c *client) Watch(ctx context.Context, list kclient.ObjectList, opts ...kclient.ListOption) (watch.Interface, error) {
	return c.backend.Watch(ctx, list, opts...)
}

func (c *client) Scheme() *runtime.Scheme {
	return c.reader.client.Scheme()
}

func (c *client) RESTMapper() meta.RESTMapper {
	return c.reader.client.RESTMapper()
}

type writer struct {
	client   kclient.Client
	registry TriggerRegistry
}

func (w *writer) DeleteAllOf(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteAllOfOption) error {
	delOpts := &kclient.DeleteAllOfOptions{}
	for _, opt := range opts {
		opt.ApplyToDeleteAllOf(delOpts)
	}
	if err := w.registry.Watch(obj, delOpts.Namespace, "", delOpts.LabelSelector, delOpts.FieldSelector); err != nil {
		return err
	}
	return w.client.DeleteAllOf(ctx, obj, opts...)
}

func (w *writer) Delete(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteOption) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return w.client.Delete(ctx, obj, opts...)
}

func (w *writer) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.PatchOption) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return w.client.Patch(ctx, obj, patch, opts...)
}

func (w *writer) Update(ctx context.Context, obj kclient.Object, opts ...kclient.UpdateOption) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return w.client.Update(ctx, obj, opts...)
}

func (w *writer) Create(ctx context.Context, obj kclient.Object, opts ...kclient.CreateOption) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return w.client.Create(ctx, obj, opts...)
}

type subResourceClient struct {
	writer   kclient.SubResourceWriter
	reader   kclient.SubResourceReader
	registry TriggerRegistry
}

type status struct {
	client   kclient.Client
	registry TriggerRegistry
}

func (s *status) Status() kclient.StatusWriter {
	return &subResourceClient{
		writer:   s.client.Status(),
		registry: s.registry,
	}
}

func (s *subResourceClient) Get(ctx context.Context, obj kclient.Object, subResource kclient.Object, opts ...kclient.SubResourceGetOption) error {
	if err := s.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}

	return s.reader.Get(ctx, obj, subResource, opts...)
}

func (s *subResourceClient) Update(ctx context.Context, obj kclient.Object, opts ...kclient.SubResourceUpdateOption) error {
	if err := s.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return s.writer.Update(ctx, obj, opts...)
}

func (s *subResourceClient) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.SubResourcePatchOption) error {
	if err := s.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return s.writer.Patch(ctx, obj, patch, opts...)
}

func (s *subResourceClient) Create(ctx context.Context, obj kclient.Object, subResource kclient.Object, opts ...kclient.SubResourceCreateOption) error {
	if err := s.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return s.writer.Create(ctx, obj, subResource, opts...)
}

type reader struct {
	scheme   *runtime.Scheme
	client   kclient.Client
	registry TriggerRegistry
}

func (a *reader) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return a.client.GroupVersionKindFor(obj)
}

func (a *reader) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return a.client.IsObjectNamespaced(obj)
}

func (a *reader) SubResource(subResource string) kclient.SubResourceClient {
	c := a.client.SubResource(subResource)
	return &subResourceClient{
		writer:   c,
		reader:   c,
		registry: a.registry,
	}
}

func (a *reader) Get(ctx context.Context, key kclient.ObjectKey, obj kclient.Object, opts ...kclient.GetOption) error {
	if err := a.registry.Watch(obj, key.Namespace, key.Name, nil, nil); err != nil {
		return err
	}

	return a.client.Get(ctx, key, obj, opts...)
}

func (a *reader) List(ctx context.Context, list kclient.ObjectList, opts ...kclient.ListOption) error {
	listOpt := &kclient.ListOptions{}
	for _, opt := range opts {
		opt.ApplyToList(listOpt)
	}

	if err := a.registry.Watch(list, listOpt.Namespace, "", listOpt.LabelSelector, listOpt.FieldSelector); err != nil {
		return err
	}

	return a.client.List(ctx, list, listOpt)
}
