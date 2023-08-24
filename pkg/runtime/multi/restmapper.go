package multi

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type multiRestMapper struct {
	multiClient
}

func (m multiRestMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	c, err := m.getClientForGroup(resource.Group)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return c.RESTMapper().KindFor(resource)
}

func (m multiRestMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	c, err := m.getClientForGroup(resource.Group)
	if err != nil {
		return nil, err
	}
	return c.RESTMapper().KindsFor(resource)
}

func (m multiRestMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	c, err := m.getClientForGroup(input.Group)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	return c.RESTMapper().ResourceFor(input)
}

func (m multiRestMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	c, err := m.getClientForGroup(input.Group)
	if err != nil {
		return nil, err
	}
	return c.RESTMapper().ResourcesFor(input)
}

func (m multiRestMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	c, err := m.getClientForGroup(gk.Group)
	if err != nil {
		return nil, err
	}
	return c.RESTMapper().RESTMapping(gk, versions...)
}

func (m multiRestMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	c, err := m.getClientForGroup(gk.Group)
	if err != nil {
		return nil, err
	}
	return c.RESTMapper().RESTMappings(gk, versions...)
}

func (m multiRestMapper) ResourceSingularizer(resource string) (string, error) {
	for _, c := range m.clients {
		if res, err := c.RESTMapper().ResourceSingularizer(resource); err == nil {
			return res, nil
		}
	}

	return m.defaultClient.RESTMapper().ResourceSingularizer(resource)
}
