package router

import (
	"sync"

	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/rancher/wrangler/pkg/merr"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func isObjectForRequest(req Request, obj meta.Object) bool {
	return obj.GetName() == req.Name &&
		obj.GetNamespace() == req.Namespace &&
		obj.GetObjectKind().GroupVersionKind() == req.GVK
}

type handlers struct {
	lock     sync.RWMutex
	handlers map[schema.GroupVersionKind][]Handler
}

func (h *handlers) GVKs() (result []schema.GroupVersionKind) {
	for gvk := range h.handlers {
		result = append(result, gvk)
	}
	return result
}

func (h *handlers) AddHandler(gvk schema.GroupVersionKind, handler Handler) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.handlers[gvk] = append(h.handlers[gvk], handler)
}

func (h *handlers) Handles(req Request) bool {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return len(h.handlers[req.GVK]) > 0
}

func (h *handlers) Handle(req Request, resp *response) error {
	h.lock.RLock()
	var (
		errs     []error
		handlers = h.handlers[req.GVK]
	)
	h.lock.RUnlock()

	i := 0
	for _, h := range handlers {
		err := h.Handle(req, resp)
		if err == nil {
			for ; i < len(resp.objects); i++ {
				if isObjectForRequest(req, resp.objects[i]) {
					req.Object = resp.objects[i]
				}
			}
		} else {
			errs = append(errs, err)
		}
	}
	return merr.NewErrors(errs...)
}
