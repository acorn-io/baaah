package multi

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type multiRestMapper struct {
	multiClient
}

func (m multiRestMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	for _, c := range m.clients {
		if res, err := c.RESTMapper().KindFor(resource); err == nil {
			return res, nil
		}
	}

	return m.defaultClient.RESTMapper().KindFor(resource)
}

func (m multiRestMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	for _, c := range m.clients {
		if res, err := c.RESTMapper().KindsFor(resource); err == nil {
			return res, nil
		}
	}

	return m.defaultClient.RESTMapper().KindsFor(resource)
}

func (m multiRestMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	for _, c := range m.clients {
		if res, err := c.RESTMapper().ResourceFor(input); err == nil {
			return res, nil
		}
	}

	return m.defaultClient.RESTMapper().ResourceFor(input)
}

func (m multiRestMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	for _, c := range m.clients {
		if res, err := c.RESTMapper().ResourcesFor(input); err == nil {
			return res, nil
		}
	}

	return m.defaultClient.RESTMapper().ResourcesFor(input)
}

func (m multiRestMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	for _, c := range m.clients {
		if res, err := c.RESTMapper().RESTMapping(gk, versions...); err == nil {
			return res, nil
		}
	}

	return m.defaultClient.RESTMapper().RESTMapping(gk, versions...)
}

func (m multiRestMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	for _, c := range m.clients {
		if res, err := c.RESTMapper().RESTMappings(gk, versions...); err == nil {
			return res, nil
		}
	}

	return m.defaultClient.RESTMapper().RESTMappings(gk, versions...)
}

func (m multiRestMapper) ResourceSingularizer(resource string) (string, error) {
	for _, c := range m.clients {
		if res, err := c.RESTMapper().ResourceSingularizer(resource); err == nil {
			return res, nil
		}
	}

	return m.defaultClient.RESTMapper().ResourceSingularizer(resource)
}
