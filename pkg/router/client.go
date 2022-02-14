package router

import (
	"context"
	"sync"

	"github.com/ibuildthecloud/baaah/pkg/backend"
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type callHistory struct {
	gvk       schema.GroupVersionKind
	namespace string
	name      string
	selector  labels.Selector
}

type client struct {
	reader
	writer
}

type writer struct {
	ctx    context.Context
	writer backend.Writer
}

func (w *writer) Delete(obj meta.Object) error {
	return w.writer.Delete(w.ctx, obj)
}

func (w *writer) Update(obj meta.Object) error {
	return w.writer.Update(w.ctx, obj)
}

func (w *writer) UpdateStatus(obj meta.Object) error {
	return w.writer.UpdateStatus(w.ctx, obj)
}

func (w *writer) Create(obj meta.Object) error {
	return w.writer.Create(w.ctx, obj)
}

type reader struct {
	ctx context.Context

	scheme           *runtime.Scheme
	reader           backend.Reader
	defaultNamespace string
	callHistory      []callHistory
	historyLock      sync.Mutex
}

func (a *reader) addCall(obj runtime.Object, ns, name string, selector labels.Selector) error {
	gvk, err := a.reader.GVKForObject(obj, a.scheme)
	if err != nil {
		return err
	}

	a.historyLock.Lock()
	defer a.historyLock.Unlock()

	a.callHistory = append(a.callHistory, callHistory{
		gvk:       gvk,
		namespace: ns,
		name:      name,
		selector:  selector,
	})
	return nil
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

	return a.addCall(obj, ns, name, nil)
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

	err := a.reader.List(a.ctx, obj, &meta.ListOptions{
		Namespace: ns,
		Selector:  sel,
	})
	if err != nil {
		return err
	}

	return a.addCall(obj, ns, "", sel)
}
