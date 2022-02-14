package router

import (
	"sync"

	"github.com/ibuildthecloud/baaah/pkg/backend"
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type triggers struct {
	lock      sync.RWMutex
	matchers  map[schema.GroupVersionKind]map[enqueueTarget]matcher
	trigger   backend.Trigger
	gvkLookup backend.Reader
	scheme    *runtime.Scheme
}

type enqueueTarget struct {
	key string
	gvk schema.GroupVersionKind
}

func (m *triggers) invokeTriggers(req Request) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for enqueueTarget, matcher := range m.matchers[req.GVK] {
		if matcher.Match(req.GVK, req.Namespace, req.Name, req.Object) {
			m.trigger.Trigger(enqueueTarget.gvk, enqueueTarget.key)
		}
	}
}

func (m *triggers) Trigger(req Request, resp *response) ([]schema.GroupVersionKind, error) {
	if !req.FromTrigger {
		m.invokeTriggers(req)
	}
	return m.watchResults(req, resp)
}

func (m *triggers) watchResults(req Request, resp *response) ([]schema.GroupVersionKind, error) {
	matcher, err := m.toMatcher(req, resp)
	if err != nil {
		return nil, err
	}
	if req.Object == nil || matcher == nil {
		m.clearMatchers(req.GVK, req.Key)
		return nil, nil
	}
	m.saveMatchers(req.GVK, req.Key, matcher)
	return matcher.WatchedGVKs(), nil
}

func (m *triggers) clearMatchers(gvk schema.GroupVersionKind, key string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	deleteKey := enqueueTarget{
		key: key,
		gvk: gvk,
	}
	for _, m := range m.matchers {
		delete(m, deleteKey)
	}
}

func (m *triggers) saveMatchers(gvk schema.GroupVersionKind, key string, mr matcher) {
	m.lock.Lock()
	defer m.lock.Unlock()

	target := enqueueTarget{
		key: key,
		gvk: gvk,
	}
	for _, sourceGVK := range mr.WatchedGVKs() {
		if target.gvk == sourceGVK {
			continue
		}
		matchers, ok := m.matchers[sourceGVK]
		if !ok {
			matchers = map[enqueueTarget]matcher{}
			m.matchers[sourceGVK] = matchers
		}
		matchers[target] = mr
	}
}

func (m *triggers) toMatcher(req Request, resp *response) (matcher, error) {
	result := matchSet{}
	for _, o := range resp.objects {
		gvk, err := m.gvkLookup.GVKForObject(o, m.scheme)
		if err != nil {
			return nil, err
		}
		result[gvk] = append(result[gvk], objectMatcher{
			Namespace: o.GetNamespace(),
			Name:      o.GetName(),
		})
	}

	c, ok := req.Client.(*client)
	if ok {
		for _, call := range c.callHistory {
			result[call.gvk] = append(result[call.gvk], objectMatcher{
				Namespace: call.namespace,
				Name:      call.name,
			})
		}
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result, nil
}

type matcher interface {
	WatchedGVKs() []schema.GroupVersionKind
	Match(gvk schema.GroupVersionKind, ns, name string, obj meta.Object) bool
}

type objectMatcher struct {
	Namespace string
	Name      string
	Selector  labels.Selector
}

func (o *objectMatcher) Match(gvk schema.GroupVersionKind, ns, name string, obj meta.Object) bool {
	if o.Name != "" {
		return o.Name == name &&
			o.Namespace == ns
	}
	if o.Namespace != "" && o.Namespace != ns {
		return false
	}
	if o.Selector != nil {
		if obj == nil {
			return false
		}
		return o.Selector.Matches(labels.Set(obj.GetLabels()))

	}
	return false
}

type matchSet map[schema.GroupVersionKind][]objectMatcher

func (m matchSet) WatchedGVKs() (result []schema.GroupVersionKind) {
	for k := range m {
		result = append(result, k)
	}
	return result
}

func (m matchSet) Match(gvk schema.GroupVersionKind, ns, name string, obj meta.Object) bool {
	for _, matcher := range m[gvk] {
		if matcher.Match(gvk, ns, name, obj) {
			return true
		}
	}
	return false
}
