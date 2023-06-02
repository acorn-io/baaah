package router

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/backend"
	"github.com/acorn-io/baaah/pkg/log"
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/moby/locker"
	"golang.org/x/exp/maps"
	"golang.org/x/time/rate"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TriggerPrefix = "_t "
	ReplayPrefix  = "_r "
)

type HandlerSet struct {
	ctx      context.Context
	name     string
	scheme   *runtime.Scheme
	backend  backend.Backend
	handlers handlers
	triggers triggers
	save     save
	onError  ErrorHandler

	watchingLock sync.Mutex
	watching     map[schema.GroupVersionKind]bool
	locker       locker.Locker

	limiterLock sync.Mutex
	limiters    map[limiterKey]*rate.Limiter
	waiting     map[limiterKey]struct{}
}

type limiterKey struct {
	key string
	gvk schema.GroupVersionKind
}

func NewHandlerSet(name string, scheme *runtime.Scheme, backend backend.Backend) *HandlerSet {
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
			apply:  apply.New(backend).WithOwnerSubContext(name),
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

func toObject(obj runtime.Object) kclient.Object {
	if obj == nil {
		return nil
	}
	// yep panic if it's not this interface
	return obj.DeepCopyObject().(kclient.Object)
}

type triggerRegistry struct {
	gvk     schema.GroupVersionKind
	gvks    map[schema.GroupVersionKind]bool
	key     string
	trigger *triggers
}

func (t *triggerRegistry) WatchingGVKs() []schema.GroupVersionKind {
	return maps.Keys(t.gvks)

}
func (t *triggerRegistry) Watch(obj runtime.Object, namespace, name string, sel labels.Selector, fields fields.Selector) error {
	gvk, ok, err := t.trigger.Register(t.gvk, t.key, obj, namespace, name, sel, fields)
	if err != nil {
		return err
	}
	if ok {
		t.gvks[gvk] = true
	}
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
				scheme:   m.scheme,
				client:   m.backend,
				registry: triggerRegistry,
			},
			writer: writer{
				client:   m.backend,
				registry: triggerRegistry,
			},
			status: status{
				client:   m.backend,
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

func (m *HandlerSet) AddHandler(objType kclient.Object, handler Handler) {
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
		if err := m.backend.Watch(m.ctx, gvk, m.name, m.onChange); err == nil {
			m.watching[gvk] = true
		} else {
			watchErrs = append(watchErrs, err)
		}
	}
	m.watchingLock.Unlock()
	return merr.NewErrors(watchErrs...)
}

func (m *HandlerSet) checkDelay(gvk schema.GroupVersionKind, key string) bool {
	m.limiterLock.Lock()
	defer m.limiterLock.Unlock()
	lKey := limiterKey{key: key, gvk: gvk}

	if _, ok := m.waiting[lKey]; ok {
		return false
	}

	limit, ok := m.limiters[lKey]
	if !ok {
		// Limit to once every 15 seconds with a burst of 10. This limits the
		// overall rate at which we can process a key regardless of the key
		// source (change event, trigger, error re-enqueue)
		limit = rate.NewLimiter(rate.Limit(1.0/15.0), 10)
		if m.limiters == nil {
			m.limiters = map[limiterKey]*rate.Limiter{}
		}
		m.limiters[lKey] = limit
	}

	delay := limit.Reserve().Delay()
	if delay > 0 {
		if m.waiting == nil {
			m.waiting = map[limiterKey]struct{}{}
		}
		m.waiting[lKey] = struct{}{}
		go func() {
			log.Warnf("Backing off [%s] [%s] for %s", key, gvk, delay)
			time.Sleep(delay)
			m.limiterLock.Lock()
			defer m.limiterLock.Unlock()
			delete(m.waiting, lKey)
			_ = m.backend.Trigger(gvk, ReplayPrefix+key, 0)
		}()
		return false
	}

	return true
}

func (m *HandlerSet) forgetBackoff(gvk schema.GroupVersionKind, key string) {
	m.limiterLock.Lock()
	defer m.limiterLock.Unlock()
	delete(m.limiters, limiterKey{key: key, gvk: gvk})
}

func (m *HandlerSet) onChange(gvk schema.GroupVersionKind, key string, runtimeObject runtime.Object) (runtime.Object, error) {
	fromTrigger := false
	fromReplay := false
	if strings.HasPrefix(key, TriggerPrefix) {
		fromTrigger = true
		key = strings.TrimPrefix(key, TriggerPrefix)
	}
	if strings.HasPrefix(key, ReplayPrefix) {
		fromTrigger = false
		fromReplay = true
		key = strings.TrimPrefix(key, ReplayPrefix)
	}

	if !fromReplay && !fromTrigger {
		// Process delay have key has be reassigned from the TriggerPrefix
		if !m.checkDelay(gvk, key) {
			return runtimeObject, nil
		}
	}

	obj, err := m.scheme.New(gvk)
	if err != nil {
		return nil, err
	}

	ns, name, ok := strings.Cut(key, "/")
	if !ok {
		name = key
		ns = ""
	}

	lockKey := gvk.Kind + " " + key
	m.locker.Lock(lockKey)
	defer func() { _ = m.locker.Unlock(lockKey) }()

	err = m.backend.Get(m.ctx, kclient.ObjectKey{Name: name, Namespace: ns}, obj.(kclient.Object))
	if err == nil {
		runtimeObject = obj
	} else if err != nil && !apierror.IsNotFound(err) {
		return nil, err
	}

	if runtimeObject == nil {
		m.forgetBackoff(gvk, key)
	}

	return m.handle(gvk, key, runtimeObject, fromTrigger)
}

func (m *HandlerSet) handleError(req Request, resp Response, err error) error {
	if m.onError != nil {
		return m.onError(req, resp, err)
	}
	return err
}

func (m *HandlerSet) handle(gvk schema.GroupVersionKind, key string, unmodifiedObject runtime.Object, trigger bool) (runtime.Object, error) {
	req, resp, err := m.newRequestResponse(gvk, key, unmodifiedObject, trigger)
	if err != nil {
		return nil, err
	}

	handles := m.handlers.Handles(req)
	if handles {
		if req.FromTrigger {
			log.Infof("Handling trigger [%s/%s] [%v]", req.Namespace, req.Name, req.GVK)
		} else {
			log.Infof("Handling [%s/%s] [%v]", req.Namespace, req.Name, req.GVK)
		}

		if err := m.handlers.Handle(req, resp); err != nil {
			if err := m.handleError(req, resp, err); err != nil {
				return nil, err
			}
		}
	}

	if err := m.triggers.Trigger(req, resp); err != nil {
		if err := m.handleError(req, resp, err); err != nil {
			return nil, err
		}
	}

	if handles {
		m.watchingLock.Lock()
		keys := maps.Keys(m.watching)
		m.watchingLock.Unlock()
		newObj, err := m.save.save(unmodifiedObject, req, resp, keys)
		if err != nil {
			if err := m.handleError(req, resp, err); err != nil {
				return nil, err
			}
		}
		req.Object = newObj

		if resp.delay > 0 {
			if err := m.backend.Trigger(gvk, key, resp.delay); err != nil {
				return nil, err
			}
		}
	}

	return req.Object, m.handleError(req, resp, err)
}

type response struct {
	delay       time.Duration
	objects     []kclient.Object
	registry    TriggerRegistry
	noPrune     bool
	noPruneGVKs []schema.GroupVersionKind
}

func (r *response) DisablePrune() {
	r.noPrune = true
}

func (r *response) WithoutPruneGVKs(gvks ...schema.GroupVersionKind) {
	r.noPruneGVKs = append(r.noPruneGVKs, gvks...)
}

func (r *response) RetryAfter(delay time.Duration) {
	if r.delay == 0 || delay < r.delay {
		r.delay = delay
	}
}

func (r *response) Objects(objs ...kclient.Object) {
	for _, obj := range objs {
		_ = r.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil)
		r.objects = append(r.objects, obj)
	}
}

func (r *response) WatchingGVKs() []schema.GroupVersionKind {
	return r.registry.WatchingGVKs()
}
