package router

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/acorn-io/baaah/pkg/backend"
	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/moby/locker"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/sirupsen/logrus"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const TriggerPrefix = "_t "

type HandlerSet struct {
	ctx      context.Context
	name     string
	scheme   *runtime.Scheme
	backend  backend.Backend
	handlers handlers
	triggers triggers
	save     save

	watchingLock sync.Mutex
	watching     map[schema.GroupVersionKind]bool
	locker       locker.Locker
}

func NewHandlerSet(name string, scheme *runtime.Scheme, backend backend.Backend, apply apply.Apply) *HandlerSet {
	hs := &HandlerSet{
		name:    name,
		scheme:  scheme,
		backend: backend,
		handlers: handlers{
			handlers: map[schema.GroupVersionKind][]Handler{},
		},
		triggers: triggers{
			matchers:  map[schema.GroupVersionKind]map[enqueueTarget][]matcher{},
			trigger:   backend,
			gvkLookup: backend,
			scheme:    scheme,
		},
		save: save{
			setID:  name,
			apply:  apply,
			cache:  backend,
			client: backend,
		},
		watching: map[schema.GroupVersionKind]bool{},
	}
	hs.triggers.watcher = hs
	return hs
}

func (m *HandlerSet) Start(ctx context.Context) error {
	m.ctx = ctx
	if err := m.WatchGVK(m.handlers.GVKs()...); err != nil {
		return err
	}
	return m.backend.Start(ctx)
}

func toObject(obj runtime.Object) meta.Object {
	if obj == nil {
		return nil
	}
	// yep panic if it's not this interface
	return obj.DeepCopyObject().(meta.Object)
}

type triggerRegistry struct {
	gvk     schema.GroupVersionKind
	gvks    map[schema.GroupVersionKind]bool
	key     string
	trigger *triggers
}

func (t *triggerRegistry) WatchingGVKs() []schema.GroupVersionKind {
	return typed.Keys(t.gvks)

}
func (t *triggerRegistry) Watch(obj runtime.Object, namespace, name string, sel labels.Selector) error {
	gvk, err := t.trigger.Register(t.gvk, t.key, obj, namespace, name, sel)
	if err != nil {
		return err
	}
	t.gvks[gvk] = true
	return nil
}

func (m *HandlerSet) newRequestResponse(gvk schema.GroupVersionKind, key string, runtimeObject runtime.Object, trigger bool) (Request, *response, error) {
	var (
		obj = toObject(runtimeObject)
	)

	ns, name, ok := strings.Cut(key, "/")
	if !ok {
		name = key
		ns = ""
	}

	triggerRegistry := &triggerRegistry{
		gvk:     gvk,
		key:     key,
		trigger: &m.triggers,
		gvks:    map[schema.GroupVersionKind]bool{},
	}

	resp := response{
		registry: triggerRegistry,
	}

	req := Request{
		FromTrigger: trigger,
		Client: &client{
			reader: reader{
				ctx:              m.ctx,
				scheme:           m.scheme,
				reader:           m.backend,
				defaultNamespace: ns,
				registry:         triggerRegistry,
			},
			writer: writer{
				ctx:      m.ctx,
				writer:   m.backend,
				registry: triggerRegistry,
			},
		},
		Ctx:       m.ctx,
		GVK:       gvk,
		Object:    obj,
		Namespace: ns,
		Name:      name,
		Key:       key,
	}

	return req, &resp, nil
}

func (m *HandlerSet) AddHandler(objType meta.Object, handler Handler) {
	gvk, err := m.backend.GVKForObject(objType, m.scheme)
	if err != nil {
		panic(fmt.Sprintf("scheme does not know gvk for %T", objType))
	}
	m.handlers.AddHandler(gvk, handler)
}

func (m *HandlerSet) WatchGVK(gvks ...schema.GroupVersionKind) error {
	var watchErrs []error
	m.watchingLock.Lock()
	for _, gvk := range gvks {
		if m.watching[gvk] {
			continue
		}
		if err := m.backend.Watch(m.ctx, gvk, m.name, m.onChange); err != nil {
			watchErrs = append(watchErrs, err)
		}
		m.watching[gvk] = true
	}
	m.watchingLock.Unlock()
	return merr.NewErrors(watchErrs...)
}

func (m *HandlerSet) onChange(gvk schema.GroupVersionKind, key string, runtimeObject runtime.Object) (runtime.Object, error) {
	fromTrigger := false
	if strings.HasPrefix(key, TriggerPrefix) {
		fromTrigger = true
		key = strings.TrimPrefix(key, TriggerPrefix)

		obj, err := m.scheme.New(gvk)
		if err != nil {
			return nil, err
		}

		ns, name, ok := strings.Cut(key, "/")
		if !ok {
			name = key
			ns = ""
		}

		err = m.backend.Get(m.ctx, obj.(meta.Object), name, &meta.GetOptions{
			Namespace: ns,
		})
		if err == nil {
			runtimeObject = obj
		} else if err != nil && !apierror.IsNotFound(err) {
			return nil, err
		}
	}
	return m.handle(gvk, key, runtimeObject, fromTrigger)
}

func (m *HandlerSet) handle(gvk schema.GroupVersionKind, key string, unmodifiedObject runtime.Object, trigger bool) (runtime.Object, error) {
	req, resp, err := m.newRequestResponse(gvk, key, unmodifiedObject, trigger)
	if err != nil {
		return nil, err
	}

	m.locker.Lock(req.GVK.Kind + req.Key)
	defer m.locker.Unlock(req.GVK.Kind + req.Key)

	if req.FromTrigger {
		logrus.Infof("Trigger %s/%s %v", req.Namespace, req.Name, req.GVK)
	} else {
		logrus.Infof("Handling %s/%s %v", req.Namespace, req.Name, req.GVK)
	}

	handles := m.handlers.Handles(req)
	if handles {
		if err := m.handlers.Handle(req, resp); err != nil {
			return nil, err
		}
	}

	if err := m.triggers.Trigger(req, resp); err != nil {
		return nil, err
	}

	if handles {
		newObj, err := m.save.save(unmodifiedObject, req, resp, typed.Keys(m.watching))
		if err != nil {
			return nil, err
		}
		req.Object = newObj
	}

	return req.Object, nil
}

type response struct {
	delay    time.Duration
	objects  []meta.Object
	registry TriggerRegistry
}

func (r *response) RetryAfter(delay time.Duration) {
	if r.delay == 0 || delay < r.delay {
		r.delay = delay
	}
}

func (r *response) Objects(objs ...meta.Object) {
	for _, obj := range objs {
		r.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil)
		r.objects = append(r.objects, obj)
	}
}

func (r *response) WatchingGVKs() []schema.GroupVersionKind {
	return r.registry.WatchingGVKs()
}
