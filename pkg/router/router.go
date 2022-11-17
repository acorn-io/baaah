package router

import (
	"context"

	"github.com/acorn-io/baaah/pkg/backend"
	"k8s.io/apimachinery/pkg/labels"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Router struct {
	RouteBuilder

	OnErrorHandler ErrorHandler
	handlers       *HandlerSet
}

func New(handlerSet *HandlerSet) *Router {
	r := &Router{
		handlers: handlerSet,
	}
	r.RouteBuilder.router = r
	return r
}

func (r *Router) Backend() backend.Backend {
	return r.handlers.backend
}

type RouteBuilder struct {
	includeRemove bool
	finalizeID    string
	router        *Router
	objType       kclient.Object
	name          string
	namespace     string
	middleware    []Middleware
	sel           labels.Selector
}

func (r RouteBuilder) Middleware(m ...Middleware) RouteBuilder {
	r.middleware = append(r.middleware, m...)
	return r
}

func (r RouteBuilder) Namespace(namespace string) RouteBuilder {
	r.namespace = namespace
	return r
}

func (r RouteBuilder) Selector(sel labels.Selector) RouteBuilder {
	r.sel = sel
	return r
}

func (r RouteBuilder) Name(name string) RouteBuilder {
	r.name = name
	return r
}

func (r RouteBuilder) IncludeRemoved() RouteBuilder {
	r.includeRemove = true
	return r
}

func (r RouteBuilder) Finalize(finalizerID string, h Handler) {
	r.finalizeID = finalizerID
	r.Handler(h)
}

func (r RouteBuilder) FinalizeFunc(finalizerID string, h HandlerFunc) {
	r.finalizeID = finalizerID
	r.Handler(h)
}

func (r RouteBuilder) Type(objType kclient.Object) RouteBuilder {
	r.objType = objType
	return r
}

func (r RouteBuilder) HandlerFunc(h HandlerFunc) {
	r.Handler(h)
}

func (r RouteBuilder) Handler(h Handler) {
	result := h
	if r.finalizeID != "" {
		result = FinalizerHandler{
			FinalizerID: r.finalizeID,
			Next:        result,
		}
	}
	for i := len(r.middleware) - 1; i >= 0; i-- {
		result = r.middleware[i](result)
	}
	if r.name != "" || r.namespace != "" {
		result = NameNamespaceFilter{
			Next:      result,
			Name:      r.name,
			Namespace: r.namespace,
		}
	}
	if r.sel != nil {
		result = SelectorFilter{
			Next:     result,
			Selector: r.sel,
		}
	}
	if !r.includeRemove && r.finalizeID == "" {
		result = IgnoreRemoveHandler{
			Next: result,
		}
	}

	r.router.handlers.AddHandler(r.objType, result)
}

func (r *Router) Start(ctx context.Context) error {
	r.handlers.onError = r.OnErrorHandler
	return r.handlers.Start(ctx)
}

func (r *Router) Handle(objType kclient.Object, h Handler) {
	r.RouteBuilder.Type(objType).Handler(h)
}

func (r *Router) HandleFunc(objType kclient.Object, h HandlerFunc) {
	r.RouteBuilder.Type(objType).Handler(h)
}

type IgnoreRemoveHandler struct {
	Next Handler
}

func (i IgnoreRemoveHandler) Handle(req Request, resp Response) error {
	if req.Object == nil || !req.Object.GetDeletionTimestamp().IsZero() {
		return nil
	}
	return i.Next.Handle(req, resp)
}

type NameNamespaceFilter struct {
	Next      Handler
	Name      string
	Namespace string
}

func (n NameNamespaceFilter) Handle(req Request, resp Response) error {
	if n.Name != "" && req.Name != n.Name {
		return nil
	}
	if n.Namespace != "" && req.Namespace != n.Namespace {
		return nil
	}
	return n.Next.Handle(req, resp)
}

type SelectorFilter struct {
	Next     Handler
	Selector labels.Selector
}

func (s SelectorFilter) Handle(req Request, resp Response) error {
	if req.Object == nil || !s.Selector.Matches(labels.Set(req.Object.GetLabels())) {
		return nil
	}
	return s.Next.Handle(req, resp)
}
