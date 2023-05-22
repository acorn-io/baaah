package router

import (
	"context"

	apierror "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetOrCreate(ctx context.Context, client kclient.Client, obj kclient.Object, prepare func() error) error {
	ns, name := obj.GetNamespace(), obj.GetName()
	err := client.Get(ctx, kclient.ObjectKey{
		Namespace: ns,
		Name:      name,
	}, obj)
	if apierror.IsNotFound(err) {
		if prepare != nil {
			err := prepare()
			if err != nil {
				return err
			}
		}
		err = client.Create(ctx, obj)
		if apierror.IsAlreadyExists(err) {
			return client.Get(ctx, kclient.ObjectKey{
				Namespace: ns,
				Name:      name,
			}, obj)
		}
	}
	return err
}
