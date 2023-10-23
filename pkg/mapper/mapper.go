package mapper

import (
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

var (
	gvrCache    = map[schema.GroupVersionResource][]schema.GroupVersionKind{}
	gvrGvrCache = map[schema.GroupVersionResource][]schema.GroupVersionResource{}
	gkCache     = map[gkKey][]*meta.RESTMapping{}
	nameCache   = map[string]string{}
	cacheLock   sync.RWMutex
)

type gkKey struct {
	versions string
	gk       schema.GroupKind
}

func New(cfg *rest.Config) (meta.RESTMapper, error) {
	disc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &RESTMapperGlobalCache{
		disc: disc,
	}, nil
}

type RESTMapperGlobalCache struct {
	disc discovery.DiscoveryInterface
	api  []*restmapper.APIGroupResources
}

func (m *RESTMapperGlobalCache) withClient(f func(m meta.RESTMapper) error) error {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	shouldRetry := true
	if len(m.api) == 0 {
		shouldRetry = false
		api, err := restmapper.GetAPIGroupResources(m.disc)
		if err != nil {
			return err
		}
		m.api = api
	}

	mapper := restmapper.NewDiscoveryRESTMapper(m.api)
	if err := f(mapper); err == nil {
		return nil
	} else {
		if !shouldRetry {
			return err
		}
	}

	api, err := restmapper.GetAPIGroupResources(m.disc)
	if err != nil {
		return err
	}
	m.api = api

	return f(restmapper.NewDiscoveryRESTMapper(m.api))
}

func (m *RESTMapperGlobalCache) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	gvk, err := m.KindsFor(resource)
	if err != nil {
		return schema.GroupVersionKind{}, nil
	}
	return gvk[0], nil
}

func (m *RESTMapperGlobalCache) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	cacheLock.RLock()
	gvks, ok := gvrCache[resource]
	cacheLock.RUnlock()

	if ok {
		return gvks, nil
	}

	logrus.Debugf("RESTMapperGlobalCache cache miss for %v", resource)

	var err error
	err = m.withClient(func(m meta.RESTMapper) error {
		gvks, err = m.KindsFor(resource)
		return err
	})
	if err != nil {
		return nil, err
	}

	cacheLock.Lock()
	gvrCache[resource] = gvks
	cacheLock.Unlock()

	return gvks, nil
}

func (m *RESTMapperGlobalCache) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	gvrs, err := m.ResourcesFor(input)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	return gvrs[0], nil
}

func (m *RESTMapperGlobalCache) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	cacheLock.RLock()
	gvrs, ok := gvrGvrCache[input]
	cacheLock.RUnlock()

	if ok {
		return gvrs, nil
	}

	logrus.Debugf("RESTMapperGlobalCache cache miss for %v", input)

	var err error
	err = m.withClient(func(m meta.RESTMapper) error {
		gvrs, err = m.ResourcesFor(input)
		return err
	})
	if err != nil {
		return nil, err
	}

	cacheLock.Lock()
	gvrGvrCache[input] = gvrs
	cacheLock.Unlock()

	return gvrs, nil
}

func (m *RESTMapperGlobalCache) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	mappings, err := m.RESTMappings(gk, versions...)
	if err != nil {
		return nil, err
	}
	return mappings[0], nil
}

func (m *RESTMapperGlobalCache) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	key := gkKey{
		versions: strings.Join(versions, "|"),
		gk:       gk,
	}
	cacheLock.RLock()
	mappings, ok := gkCache[key]
	cacheLock.RUnlock()

	if ok {
		return mappings, nil
	}

	logrus.Debugf("RESTMapperGlobalCache cache miss for %v, %v", gk, versions)

	var err error
	err = m.withClient(func(m meta.RESTMapper) error {
		mappings, err = m.RESTMappings(gk)
		return err
	})
	if err != nil {
		return nil, err
	}

	cacheLock.Lock()
	gkCache[key] = mappings
	cacheLock.Unlock()

	return mappings, nil
}

func (m *RESTMapperGlobalCache) ResourceSingularizer(resource string) (string, error) {
	cacheLock.RLock()
	singular, ok := nameCache[resource]
	cacheLock.RUnlock()

	if ok {
		return singular, nil
	}

	logrus.Debugf("RESTMapperGlobalCache cache miss for %s", resource)

	var err error
	err = m.withClient(func(m meta.RESTMapper) error {
		singular, err = m.ResourceSingularizer(resource)
		return err
	})
	if err != nil {
		return "", err
	}

	cacheLock.Lock()
	nameCache[resource] = singular
	cacheLock.Unlock()

	return singular, nil
}
