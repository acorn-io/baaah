package apply

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *apply) create(obj kclient.Object) (kclient.Object, error) {
	return obj, a.client.Create(a.ctx, obj)
}

func (a *apply) get(gvk schema.GroupVersionKind, namespace, name string) (kclient.Object, error) {
	ustr := &unstructured.Unstructured{}
	ustr.SetGroupVersionKind(gvk)
	return ustr, a.client.Get(a.ctx, kclient.ObjectKey{Namespace: namespace, Name: name}, ustr)
}

func (a *apply) delete(gvk schema.GroupVersionKind, namespace, name string) error {
	ustr := &unstructured.Unstructured{}
	ustr.SetGroupVersionKind(gvk)
	ustr.SetName(name)
	ustr.SetNamespace(namespace)
	return a.client.Delete(a.ctx, ustr)
}
