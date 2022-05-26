package router

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type matcher interface {
	Match(gvk schema.GroupVersionKind, ns, name string, obj kclient.Object) bool
	Equals(m matcher) bool
}

type objectMatcher struct {
	Namespace string
	Name      string
	Selector  labels.Selector
}

func (o *objectMatcher) Equals(other matcher) bool {
	otherMatcher, ok := other.(*objectMatcher)
	if !ok {
		return false
	}
	if o.Name != otherMatcher.Name {
		return false
	}
	if o.Namespace != otherMatcher.Namespace {
		return false
	}
	if (o.Selector == nil) != (otherMatcher.Selector == nil) {
		return false
	}
	if o.Selector != nil && o.Selector.String() != otherMatcher.Selector.String() {
		return false
	}
	return true
}

func (o *objectMatcher) Match(gvk schema.GroupVersionKind, ns, name string, obj kclient.Object) bool {
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
	return o.Namespace == ns
}
