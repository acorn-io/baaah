package baaah

import (
	"os"
	"path/filepath"

	"github.com/acorn-io/baaah/pkg/lasso"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/router"
	"k8s.io/apimachinery/pkg/runtime"
)

func DefaultRouter(scheme *runtime.Scheme) (*router.Router, error) {
	cfg, err := restconfig.New(scheme)
	if err != nil {
		return nil, err
	}

	runtime, err := lasso.NewRuntime(cfg, scheme)
	if err != nil {
		return nil, err
	}

	name := filepath.Base(os.Args[0])
	handlerSet := router.NewHandlerSet(name, scheme, runtime.Backend)
	return router.New(handlerSet), nil
}
