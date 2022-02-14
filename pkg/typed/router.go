package typed

import (
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
)

type PointerToMetaObject[T any] interface {
	*T
	meta.Object
}

type Request[T meta.Object] struct {
	router.Request
	Object T
}

type HandlerFunc[T meta.Object] func(req Request[T], resp router.Response) error

func Handler[T meta.Object](handler HandlerFunc[T]) (meta.Object, router.Handler) {
	return NewAs[T, meta.Object](), router.HandlerFunc(func(req router.Request, resp router.Response) error {
		newRequest := Request[T]{
			Request: req,
		}
		if req.Object != nil {
			newRequest.Object = req.Object.(T)
		}
		return handler(newRequest, resp)
	})
}
