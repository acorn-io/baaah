package other

import (
	"context"

	"github.com/rancher/lasso/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type List[T any] struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []T `json:"items"`
}

func (l *List[T]) DeepCopyObject() runtime.Object {
	panic("not implemented")
}

type Object interface {
	runtime.Object
	metav1.Object
}

type ObjectList interface {
	metav1.ListInterface
	runtime.Object
}

type PObject[T any] interface {
	*T
	Object
}

type Client[T any, TP PObject[T]] interface {
	Create(TP) (TP, error)
	Update(TP) (TP, error)
	UpdateStatus(TP) (TP, error)
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	Get(namespace, name string, options metav1.GetOptions) (TP, error)
	List(namespace string, opts metav1.ListOptions) (List[T], error)
	Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error)
	Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result TP, err error)
}

func NewClient[T any, TP PObject[T]](client *client.Client) Client[T, TP] {
	return &objClient[T, TP]{
		client: client,
	}
}

type objClient[T any, TP PObject[T]] struct {
	client *client.Client
}

func (c *objClient[T, TP]) Create(obj TP) (TP, error) {
	var empty T
	result := TP(&empty)
	return result, c.client.Create(context.TODO(), obj.GetNamespace(), obj, result, metav1.CreateOptions{})
}

func (c *objClient[T, TP]) Update(obj TP) (TP, error) {
	var empty T
	result := TP(&empty)
	return result, c.client.Update(context.TODO(), obj.GetNamespace(), obj, result, metav1.UpdateOptions{})
}

func (c *objClient[T, TP]) UpdateStatus(obj TP) (TP, error) {
	var empty T
	result := TP(&empty)
	return result, c.client.UpdateStatus(context.TODO(), obj.GetNamespace(), obj, result, metav1.UpdateOptions{})
}

func (c *objClient[T, TP]) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), namespace, name, *options)
}

func (c *objClient[T, TP]) Get(namespace, name string, options metav1.GetOptions) (TP, error) {
	var empty T
	result := TP(&empty)
	return result, c.client.Get(context.TODO(), namespace, name, result, options)
}

func (c *objClient[T, TP]) List(namespace string, opts metav1.ListOptions) (List[T], error) {
	result := &List[T]{}
	return *result, c.client.List(context.TODO(), namespace, result, opts)
}

func (c *objClient[T, TP]) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), namespace, opts)
}

func (c *objClient[T, TP]) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (TP, error) {
	var empty T
	result := TP(&empty)
	return result, c.client.Patch(context.TODO(), namespace, name, pt, data, result, metav1.PatchOptions{}, subresources...)
}
