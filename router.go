package baaah

import (
	"os"
	"path/filepath"

	"github.com/acorn-io/baaah/pkg/lasso"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/router"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

func DefaultRouter(scheme *runtime.Scheme) (*router.Router, error) {
	cfg, err := restconfig.New(scheme)
	if err != nil {
		return nil, err
	}

	return NewRouter(DefaultHandlerName(), "", cfg, scheme)
}

func DefaultHandlerName() string {
	return filepath.Base(os.Args[0])
}

func NewRouter(handlerName, namespace string, cfg *rest.Config, scheme *runtime.Scheme) (*router.Router, error) {
	if handlerName == "" {
		handlerName = DefaultHandlerName()
	}

	runtime, err := lasso.NewRuntimeForNamespace(cfg, namespace, scheme)
	if err != nil {
		return nil, err
	}

	handlerSet := router.NewHandlerSet(handlerName, scheme, runtime.Backend)
	return router.New(handlerSet), nil
}
