package router

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type AddToSchemer func(s *runtime.Scheme) error

type Handler interface {
	Handle(req Request, resp Response) error
}

type Middleware func(h Handler) Handler

type HandlerFunc func(req Request, resp Response) error

func (h HandlerFunc) Handle(req Request, resp Response) error {
	return h(req, resp)
}

type Request struct {
	Client      kclient.Client
	Object      kclient.Object
	Ctx         context.Context
	GVK         schema.GroupVersionKind
	Namespace   string
	Name        string
	Key         string
	FromTrigger bool
}

func (r *Request) List(object kclient.ObjectList, opts *kclient.ListOptions) error {
	return r.Client.List(r.Ctx, object, opts)
}

func (r *Request) Get(object kclient.Object, namespace, name string) error {
	return r.Client.Get(r.Ctx, Key(namespace, name), object)
}

type Response interface {
	RetryAfter(delay time.Duration)
	Objects(obj ...kclient.Object)
}

func Key(namespace, name string) kclient.ObjectKey {
	return kclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
}
