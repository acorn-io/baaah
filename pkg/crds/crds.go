package crds

import (
	"context"

	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/rancher/wrangler/pkg/crd"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Create(ctx context.Context, scheme *runtime.Scheme, gvs ...schema.GroupVersion) error {
	var wranglerCRDs []crd.CRD

	for _, gv := range gvs {
		for kind := range scheme.KnownTypes(gv) {
			gvk := gv.WithKind(kind)
			obj, err := scheme.New(gvk)
			if err != nil {
				return err
			}
			_, isObj := obj.(meta.Object)
			_, isListObj := obj.(meta.ObjectList)
			if isObj && !isListObj {
				wranglerCRDs = append(wranglerCRDs, crd.CRD{
					GVK:          gvk,
					SchemaObject: obj,
					Status:       true,
				}.WithColumnsFromStruct(obj))
			}
		}
	}

	restConfig, err := restconfig.New(scheme)
	if err != nil {
		return err
	}

	factory, err := crd.NewFactoryFromClient(restConfig)
	if err != nil {
		return err
	}

	return factory.BatchCreateCRDs(ctx, wranglerCRDs...).BatchWait()
}
