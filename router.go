package baaah

import (
	"github.com/acorn-io/baaah/pkg/lasso"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/router"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// DefaultRouter The routerName is important as this name will be used to assign ownership of objects
// created by this router. Specifically the routerName is assigned to the sub-context in the
// apply actions.
func DefaultRouter(routerName string, scheme *runtime.Scheme) (*router.Router, error) {
	cfg, err := restconfig.New(scheme)
	if err != nil {
		return nil, err
	}

	return NewRouter(routerName, "", cfg, scheme)
}

func NewRouter(handlerName, namespace string, cfg *rest.Config, scheme *runtime.Scheme) (*router.Router, error) {
	runtime, err := lasso.NewRuntimeForNamespace(cfg, namespace, scheme)
	if err != nil {
		return nil, err
	}

	handlerSet := router.NewHandlerSet(handlerName, scheme, runtime.Backend)
	return router.New(handlerSet), nil
}
