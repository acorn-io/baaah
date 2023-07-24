package multi

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewClientNotFoundError(group string) error {
	return &ClientNotFoundError{group: group}
}

type ClientNotFoundError struct {
	group string
}

func (c *ClientNotFoundError) Error() string {
	return fmt.Sprintf("client for group %s not found", c.group)
}

type multiClient struct {
	defaultClient kclient.WithWatch
	clients       map[string]kclient.WithWatch
	scheme        *runtime.Scheme
}

type clientWithFakeWatch struct {
	kclient.Client
}

func (c *clientWithFakeWatch) Watch(context.Context, kclient.ObjectList, ...kclient.ListOption) (watch.Interface, error) {
	return watch.NewFake(), nil
}

// NewClient returns a client that will use the client for the API groups it knows about.
// The default client is used for any unspecified API groups. If a client cannot be found, and a default doesn't exist,
// then every method will return a ClientNotFound error.
func NewClient(defaultClient kclient.Client, clients map[string]kclient.Client) kclient.Client {
	fakeWatchClients := make(map[string]kclient.WithWatch, len(clients))

	for group, c := range clients {
		fakeWatchClients[group] = &clientWithFakeWatch{c}
	}
	return NewWithWatch(&clientWithFakeWatch{defaultClient}, fakeWatchClients)
}

// NewWithWatch returns a client WithWatch that will use the client for the API groups it knows about.
// The default client is used for any unspecified API groups. If a client cannot be found, and a default doesn't exist,
// then every method will return a ClientNotFound error.
func NewWithWatch(defaultClient kclient.WithWatch, clients map[string]kclient.WithWatch) kclient.WithWatch {
	newScheme := runtime.NewScheme()
	gvksSeen := make(map[schema.GroupVersionKind]struct{})
	groups := make(map[string]struct{})
	for group := range clients {
		groups[group] = struct{}{}
	}

	for group, c := range clients {
		_, inGroups := groups[group]
		for key, val := range c.Scheme().AllKnownTypes() {
			if _, ok := gvksSeen[key]; !ok && inGroups && key.Group == group {
				newScheme.AddKnownTypeWithName(key, reflect.New(val).Interface().(runtime.Object))
				gvksSeen[key] = struct{}{}
			}
		}
	}

	for key, val := range defaultClient.Scheme().AllKnownTypes() {
		if _, ok := gvksSeen[key]; !ok {
			newScheme.AddKnownTypeWithName(key, reflect.New(val).Interface().(runtime.Object))
			gvksSeen[key] = struct{}{}
		}
	}

	return &multiClient{
		defaultClient: defaultClient,
		clients:       clients,
		scheme:        newScheme,
	}
}

func (m multiClient) Get(ctx context.Context, key kclient.ObjectKey, obj kclient.Object, opts ...kclient.GetOption) error {
	c, err := m.getClient(obj)
	if err != nil {
		return err
	}
	return c.Get(ctx, key, obj, opts...)
}

func (m multiClient) List(ctx context.Context, list kclient.ObjectList, opts ...kclient.ListOption) error {
	c, err := m.getClient(list)
	if err != nil {
		return err
	}
	return c.List(ctx, list, opts...)
}

func (m multiClient) Create(ctx context.Context, obj kclient.Object, opts ...kclient.CreateOption) error {
	c, err := m.getClient(obj)
	if err != nil {
		return err
	}
	return c.Create(ctx, obj, opts...)
}

func (m multiClient) Delete(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteOption) error {
	c, err := m.getClient(obj)
	if err != nil {
		return err
	}
	return c.Delete(ctx, obj, opts...)
}

func (m multiClient) Update(ctx context.Context, obj kclient.Object, opts ...kclient.UpdateOption) error {
	c, err := m.getClient(obj)
	if err != nil {
		return err
	}
	return c.Update(ctx, obj, opts...)
}

func (m multiClient) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.PatchOption) error {
	c, err := m.getClient(obj)
	if err != nil {
		return err
	}
	return c.Patch(ctx, obj, patch, opts...)
}

func (m multiClient) DeleteAllOf(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteAllOfOption) error {
	c, err := m.getClient(obj)
	if err != nil {
		return err
	}
	return c.DeleteAllOf(ctx, obj, opts...)
}

func (m multiClient) Status() kclient.SubResourceWriter {
	return statusMultiClient{m}
}

func (m multiClient) SubResource(subResource string) kclient.SubResourceClient {
	return subResourceMultiClient{m, subResource}
}

func (m multiClient) Scheme() *runtime.Scheme {
	return m.scheme
}

func (m multiClient) RESTMapper() meta.RESTMapper {
	return multiRestMapper{m}
}

func (m multiClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return apiutil.GVKForObject(obj, m.scheme)
}

func (m multiClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	c, err := m.getClient(obj)
	if err != nil {
		return false, err
	}
	return c.IsObjectNamespaced(obj)
}

func (m multiClient) Watch(ctx context.Context, obj kclient.ObjectList, opts ...kclient.ListOption) (watch.Interface, error) {
	c, err := m.getClient(obj)
	if err != nil {
		return nil, err
	}
	return c.Watch(ctx, obj, opts...)
}

func (m multiClient) getClient(obj runtime.Object) (kclient.WithWatch, error) {
	gvk, err := apiutil.GVKForObject(obj, m.scheme)
	if err != nil {
		return nil, err
	}
	return m.getClientForGroup(gvk.Group)
}

func (m multiClient) getClientForGroup(group string) (kclient.WithWatch, error) {
	if c, ok := m.clients[group]; ok {
		return c, nil
	} else if m.defaultClient != nil {
		return m.defaultClient, nil
	}

	return nil, NewClientNotFoundError(group)
}
