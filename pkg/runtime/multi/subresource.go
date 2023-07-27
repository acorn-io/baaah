package multi

import (
	"context"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type subResourceMultiClient struct {
	multiClient
	subResource string
}

func (s subResourceMultiClient) Get(ctx context.Context, obj kclient.Object, subResource kclient.Object, opts ...kclient.SubResourceGetOption) error {
	c, err := s.getClient(obj)
	if err != nil {
		return err
	}
	return c.SubResource(s.subResource).Get(ctx, obj, subResource, opts...)
}

func (s subResourceMultiClient) Create(ctx context.Context, obj kclient.Object, subResource kclient.Object, opts ...kclient.SubResourceCreateOption) error {
	c, err := s.getClient(obj)
	if err != nil {
		return err
	}
	return c.SubResource(s.subResource).Create(ctx, obj, subResource, opts...)
}

func (s subResourceMultiClient) Update(ctx context.Context, obj kclient.Object, opts ...kclient.SubResourceUpdateOption) error {
	c, err := s.getClient(obj)
	if err != nil {
		return err
	}
	return c.SubResource(s.subResource).Update(ctx, obj, opts...)
}

func (s subResourceMultiClient) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.SubResourcePatchOption) error {
	c, err := s.getClient(obj)
	if err != nil {
		return err
	}
	return c.SubResource(s.subResource).Patch(ctx, obj, patch, opts...)
}
