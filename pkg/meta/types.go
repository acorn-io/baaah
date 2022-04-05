package meta

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

type Object interface {
	runtime.Object
	metav1.Object
}

type ObjectList interface {
	metav1.ListInterface
	runtime.Object
}

type GetOptions struct {
	Namespace string
}

func (g *GetOptions) GetNamespace(defaultValue string) string {
	if g == nil || g.Namespace == "" {
		return defaultValue
	}
	return g.Namespace
}

type ListOptions struct {
	Namespace string
	Selector  labels.Selector
}

func (l *ListOptions) GetSelector() labels.Selector {
	if l == nil {
		return nil
	}
	return l.Selector
}

func (l *ListOptions) GetNamespace(defaultValue string) string {
	if l == nil || l.Namespace == "" {
		return defaultValue
	}
	return l.Namespace
}
