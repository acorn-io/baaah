package multi

import (
	"context"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type statusMultiClient struct {
	multiClient
}

func (s statusMultiClient) Create(ctx context.Context, obj, subResource kclient.Object, opts ...kclient.SubResourceCreateOption) error {
	c, err := s.getClient(obj)
	if err != nil {
		return err
	}
	return c.Status().Create(ctx, obj, subResource, opts...)
}

func (s statusMultiClient) Update(ctx context.Context, obj kclient.Object, opts ...kclient.SubResourceUpdateOption) error {
	c, err := s.getClient(obj)
	if err != nil {
		return err
	}
	return c.Status().Update(ctx, obj, opts...)
}

func (s statusMultiClient) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.SubResourcePatchOption) error {
	c, err := s.getClient(obj)
	if err != nil {
		return err
	}
	return c.Status().Patch(ctx, obj, patch, opts...)
}
