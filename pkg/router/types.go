package router

import (
	"context"
	"time"

	"github.com/acorn-io/baaah/pkg/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	Client      Client
	Object      meta.Object
	Ctx         context.Context
	GVK         schema.GroupVersionKind
	Namespace   string
	Name        string
	Key         string
	FromTrigger bool
}

type Client interface {
	Reader
	Writer
}

type Reader interface {
	Getter
	Lister
}

type Writer interface {
	Delete(obj meta.Object) error
	Update(obj meta.Object) error
	UpdateStatus(obj meta.Object) error
	Create(obj meta.Object) error
}

type Getter interface {
	Get(obj meta.Object, name string, opts *meta.GetOptions) error
}

type Lister interface {
	List(obj meta.ObjectList, opts *meta.ListOptions) error
}

type Response interface {
	RetryAfter(delay time.Duration)
	Objects(obj ...meta.Object)
}
