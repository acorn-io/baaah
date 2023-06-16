package router

import (
	"strings"
	"sync"

	"github.com/acorn-io/baaah/pkg/backend"
	"github.com/acorn-io/baaah/pkg/log"
	"github.com/acorn-io/baaah/pkg/uncached"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type triggers struct {
	lock      sync.RWMutex
	matchers  map[schema.GroupVersionKind]map[enqueueTarget][]objectMatcher
	trigger   backend.Trigger
	gvkLookup backend.Backend
	scheme    *runtime.Scheme
	watcher   watcher
}

type watcher interface {
	WatchGVK(gvks ...schema.GroupVersionKind) error
}

type enqueueTarget struct {
	key string
	gvk schema.GroupVersionKind
}

func (m *triggers) invokeTriggers(req Request) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for enqueueTarget, matchers := range m.matchers[req.GVK] {
		if enqueueTarget.gvk == req.GVK &&
			enqueueTarget.key == req.Key {
			continue
		}
		for _, matcher := range matchers {
			if matcher.Match(req.Namespace, req.Name, req.Object) {
				log.Infof("Triggering [%s] [%v] from [%s] [%v]", enqueueTarget.key, enqueueTarget.gvk, req.Key, req.GVK)
				_ = m.trigger.Trigger(enqueueTarget.gvk, enqueueTarget.key, 0)
				break
			}
		}
	}
}

func (m *triggers) register(gvk schema.GroupVersionKind, key string, targetGVK schema.GroupVersionKind, mr objectMatcher) {
	m.lock.Lock()
	defer m.lock.Unlock()

	target := enqueueTarget{
		key: key,
		gvk: gvk,
	}
	matchers, ok := m.matchers[targetGVK]
	if !ok {
		matchers = map[enqueueTarget][]objectMatcher{}
		m.matchers[targetGVK] = matchers
	}
	for _, existing := range matchers[target] {
		if existing.Equals(mr) {
			return
		}
	}
	matchers[target] = append(matchers[target], mr)
}

func (m *triggers) Trigger(req Request) {
	if !req.FromTrigger {
		m.invokeTriggers(req)
	}
}

func (m *triggers) Register(sourceGVK schema.GroupVersionKind, key string, obj runtime.Object, namespace, name string, selector labels.Selector, fields fields.Selector) (schema.GroupVersionKind, bool, error) {
	if uncached.IsWrapped(obj) {
		return schema.GroupVersionKind{}, false, nil
	}
	gvk, err := m.gvkLookup.GVKForObject(obj, m.scheme)
	if err != nil {
		return gvk, false, err
	}

	if _, ok := obj.(kclient.ObjectList); ok {
		gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	}

	m.register(sourceGVK, key, gvk, objectMatcher{
		Namespace: namespace,
		Name:      name,
		Selector:  selector,
		Fields:    fields,
	})

	return gvk, true, m.watcher.WatchGVK(gvk)
}

// UnregisterAndTrigger will unregister all triggers for the object, both as source and target.
// If a trigger source matches the object exactly, then the trigger will be invoked.
func (m *triggers) UnregisterAndTrigger(req Request) {
	m.lock.Lock()
	defer m.lock.Unlock()

	remainingMatchers := map[schema.GroupVersionKind]map[enqueueTarget][]objectMatcher{}

	for targetGVK, matchers := range m.matchers {
		for target, mts := range matchers {
			if target.gvk == req.GVK && target.key == req.Key {
				// If the target is the GVK and key we are unregistering, then skip it
				continue
			}
			for _, mt := range mts {
				if targetGVK != req.GVK || mt.Namespace != req.Namespace || mt.Name != req.Name {
					// If the matcher matches the deleted object exactly, then skip the matcher.
					if remainingMatchers[targetGVK] == nil {
						remainingMatchers[targetGVK] = make(map[enqueueTarget][]objectMatcher)
					}
					remainingMatchers[targetGVK][target] = append(remainingMatchers[targetGVK][target], mt)
				}
				if targetGVK == req.GVK && mt.Match(req.Namespace, req.Name, req.Object) {
					log.Infof("Triggering [%s] [%v] from [%s] [%v] on delete", target.key, target.gvk, req.Key, req.GVK)
					_ = m.trigger.Trigger(target.gvk, target.key, 0)
				}
			}
		}
	}

	m.matchers = remainingMatchers
}
