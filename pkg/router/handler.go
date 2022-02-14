package router

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ibuildthecloud/baaah/pkg/backend"
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/merr"
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
}

func NewHandlerSet(name string, scheme *runtime.Scheme, backend backend.Backend, apply apply.Apply) *HandlerSet {
	return &HandlerSet{
		name:    name,
		scheme:  scheme,
		backend: backend,
		handlers: handlers{
			handlers: map[schema.GroupVersionKind][]Handler{},
		},
		triggers: triggers{
			matchers:  map[schema.GroupVersionKind]map[enqueueTarget]matcher{},
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
}

func (m *HandlerSet) Start(ctx context.Context) error {
	m.ctx = ctx
	if err := m.watchGVK(m.handlers.GVKs()...); err != nil {
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

func (m *HandlerSet) newRequest(gvk schema.GroupVersionKind, key string, runtimeObject runtime.Object) Request {
	var (
		obj         = toObject(runtimeObject)
		fromTrigger = false
	)
	if strings.HasPrefix(key, TriggerPrefix) {
		fromTrigger = true
		key = strings.TrimPrefix(key, TriggerPrefix)
	}
	ns, name, ok := strings.Cut(key, "/")
	if !ok {
		name = key
		ns = ""
	}

	return Request{
		FromTrigger: fromTrigger,
		Client: &client{
			reader: reader{
				ctx:              m.ctx,
				scheme:           m.scheme,
				reader:           m.backend,
				defaultNamespace: ns,
			},
			writer: writer{
				ctx:    m.ctx,
				writer: m.backend,
			},
		},
		Ctx:       m.ctx,
		GVK:       gvk,
		Object:    obj,
		Namespace: ns,
		Name:      name,
		Key:       key,
	}
}

func (m *HandlerSet) AddHandler(objType meta.Object, handler Handler) {
	gvk, err := m.backend.GVKForObject(objType, m.scheme)
	if err != nil {
		panic(fmt.Sprintf("scheme does not know gvk for %T", objType))
	}
	m.handlers.AddHandler(gvk, handler)
}

func (m *HandlerSet) watchGVK(gvks ...schema.GroupVersionKind) error {
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
	req := m.newRequest(gvk, key, runtimeObject)
	resp := &response{}

	handles := m.handlers.Handles(req)
	if handles {
		if err := m.handlers.Handle(req, resp); err != nil {
			return nil, err
		}
	}

	watchingGVKS, err := m.triggers.Trigger(req, resp)
	if err != nil {
		return nil, err
	}

	if err := m.watchGVK(watchingGVKS...); err != nil {
		return nil, err
	}

	if handles {
		newObj, err := m.save.save(runtimeObject, req, resp, watchingGVKS)
		if err != nil {
			return nil, err
		}

		return newObj, nil
	}

	return runtimeObject, nil
}

type response struct {
	delay   time.Duration
	objects []meta.Object
}

func (r *response) RetryAfter(delay time.Duration) {
	if r.delay == 0 || delay < r.delay {
		r.delay = delay
	}
}

func (r *response) Objects(obj ...meta.Object) {
	r.objects = append(r.objects, obj...)
}
