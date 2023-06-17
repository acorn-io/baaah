package router

import (
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type objectMatcher struct {
	Namespace string
	Name      string
	Selector  labels.Selector
	Fields    fields.Selector
}

func (o *objectMatcher) Equals(other objectMatcher) bool {
	if o.Name != other.Name {
		return false
	}
	if o.Namespace != other.Namespace {
		return false
	}
	if (o.Selector == nil) != (other.Selector == nil) {
		return false
	}
	if o.Selector != nil && o.Selector.String() != other.Selector.String() {
		return false
	}
	if (o.Fields == nil) != (other.Fields == nil) {
		return false
	}
	if o.Fields != nil && o.Fields.String() != other.Fields.String() {
		return false
	}
	return true
}

func (o *objectMatcher) Match(ns, name string, obj kclient.Object) bool {
	if o.Name != "" {
		return o.Name == name &&
			o.Namespace == ns
	}
	if o.Namespace != "" && o.Namespace != ns {
		return false
	}
	if o.Selector != nil || o.Fields != nil {
		if obj == nil {
			return false
		}
		var (
			selectorMatches = true
			fieldMatches    = true
		)
		if o.Selector != nil {
			selectorMatches = o.Selector.Matches(labels.Set(obj.GetLabels()))
		}
		if o.Fields != nil {
			if i, ok := obj.(fields.Fields); ok {
				fieldMatches = o.Fields.Matches(i)
			}
		}
		return selectorMatches && fieldMatches
	}
	if o.Fields != nil {
		if obj == nil {
			return false
		}
		if i, ok := obj.(fields.Fields); ok {
			return o.Fields.Matches(i)
		}
	}
	return o.Namespace == ns
}
