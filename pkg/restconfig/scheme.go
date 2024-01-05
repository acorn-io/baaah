package restconfig

import (
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

type AddToSchemeFunc func(scheme *runtime.Scheme) error

func MustBuildScheme(addToSchemes ...AddToSchemeFunc) (*runtime.Scheme, serializer.CodecFactory, runtime.ParameterCodec, func(*runtime.Scheme) error) {
	scheme := runtime.NewScheme()
	addToScheme := AddToSchemeFunc(func(scheme *runtime.Scheme) error {
		var errs []error
		metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
		for _, f := range addToSchemes {
			err := f(scheme)
			if err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	})

	if err := addToScheme(scheme); err != nil {
		panic(fmt.Errorf("failed to build scheme: %w", err))
	}

	return scheme,
		serializer.NewCodecFactory(scheme),
		runtime.NewParameterCodec(scheme),
		addToScheme
}
