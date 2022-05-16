package router

import (
	"context"

	"github.com/acorn-io/baaah/pkg/backend"
	"github.com/acorn-io/baaah/pkg/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type TriggerRegistry interface {
	Watch(obj runtime.Object, namespace, name string, selector labels.Selector) error
	WatchingGVKs() []schema.GroupVersionKind
}

type client struct {
	reader
	writer
}

type writer struct {
	ctx      context.Context
	writer   backend.Writer
	registry TriggerRegistry
}

func (w *writer) Delete(obj meta.Object) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil); err != nil {
		return err
	}
	return w.writer.Delete(w.ctx, obj)
}

func (w *writer) Update(obj meta.Object) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil); err != nil {
		return err
	}
	return w.writer.Update(w.ctx, obj)
}

func (w *writer) UpdateStatus(obj meta.Object) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil); err != nil {
		return err
	}
	return w.writer.UpdateStatus(w.ctx, obj)
}

func (w *writer) Create(obj meta.Object) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil); err != nil {
		return err
	}
	return w.writer.Create(w.ctx, obj)
}

type reader struct {
	ctx context.Context

	scheme           *runtime.Scheme
	reader           backend.Reader
	defaultNamespace string
	registry         TriggerRegistry
}

func (a *reader) Get(obj meta.Object, name string, opts *meta.GetOptions) error {
	ns := a.defaultNamespace
	if opts != nil && opts.Namespace != "" {
		ns = opts.Namespace
	}

	err := a.reader.Get(a.ctx, obj, name, &meta.GetOptions{
		Namespace: ns,
	})
	if err != nil {
		return err
	}

	if err := a.registry.Watch(obj, ns, name, nil); err != nil {
		return err
	}
	return nil
}

func (a *reader) List(obj meta.ObjectList, opts *meta.ListOptions) error {
	var (
		sel labels.Selector
		ns  = a.defaultNamespace
	)

	if opts != nil {
		if opts.Namespace != "" {
			ns = opts.Namespace
		}
		sel = opts.Selector
	}

	if err := a.registry.Watch(obj, ns, "", sel); err != nil {
		return err
	}

	err := a.reader.List(a.ctx, obj, &meta.ListOptions{
		Namespace: ns,
		Selector:  sel,
	})
	if err != nil {
		return err
	}

	return nil
}
