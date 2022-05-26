package tester

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rancher/wrangler/pkg/randomtoken"
	"k8s.io/apimachinery/pkg/api/errors"
	meta2 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Client struct {
	DefaultNamespace string
	Objects          []kclient.Object
	SchemeObj        *runtime.Scheme
	Created          []kclient.Object
}

func (c Client) objects() []kclient.Object {
	return append(c.Objects, c.Created...)
}

func (c *Client) Get(ctx context.Context, key kclient.ObjectKey, out kclient.Object) error {
	t := reflect.TypeOf(out)
	ns := c.DefaultNamespace
	if key.Namespace != "" {
		ns = key.Namespace
	}
	for _, obj := range c.objects() {
		if reflect.TypeOf(obj) != t {
			continue
		}
		if obj.GetName() == key.Name &&
			obj.GetNamespace() == ns {
			copy(out, obj)
			return nil
		}
	}
	return errors.NewNotFound(schema.GroupResource{
		Group:    fmt.Sprintf("Unknown group from test: %T", out),
		Resource: fmt.Sprintf("Unknown resource from test: %T", out),
	}, key.Name)
}

func copy(dest, src kclient.Object) {
	srcCopy := src.DeepCopyObject()
	reflect.Indirect(reflect.ValueOf(dest)).Set(reflect.Indirect(reflect.ValueOf(srcCopy)))
}

func (c *Client) List(ctx context.Context, objList kclient.ObjectList, opts ...kclient.ListOption) error {
	listOpts := &kclient.ListOptions{}
	for _, opt := range opts {
		opt.ApplyToList(listOpts)
	}

	gvk, err := apiutil.GVKForObject(objList, c.SchemeObj)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(gvk.Kind, "List") {
		return fmt.Errorf("invalid list object %v, Kind must end with List", gvk)
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	genericObj, err := c.SchemeObj.New(gvk)
	if err != nil {
		return err
	}
	obj := genericObj.(kclient.Object)
	t := reflect.TypeOf(obj)
	ns := c.DefaultNamespace
	if listOpts.Namespace != "" {
		ns = listOpts.Namespace
	}
	var resultObjs []runtime.Object
	for _, testObj := range c.objects() {
		if testObj.GetNamespace() != ns {
			continue
		}
		if reflect.TypeOf(obj) != t {
			continue
		}
		if opts != nil && listOpts.LabelSelector != nil && !listOpts.LabelSelector.Matches(labels.Set(testObj.GetLabels())) {
			continue
		}
		copy(obj, testObj)
		if err != nil {
			return err
		}

		resultObjs = append(resultObjs, testObj)
		newObj, err := c.SchemeObj.New(gvk)
		if err != nil {
			return err
		}
		obj = newObj.(kclient.Object)
	}
	return meta2.SetList(objList, resultObjs)
}

func (c *Client) Create(ctx context.Context, obj kclient.Object, opts ...kclient.CreateOption) error {
	obj.SetUID(types.UID(uuid.New().String()))
	if obj.GetName() == "" && obj.GetGenerateName() != "" {
		r, err := randomtoken.Generate()
		if err != nil {
			return err
		}
		obj.SetName(obj.GetGenerateName() + r[:5])
	}
	c.Created = append(c.Created, obj)
	return nil
}

type Response struct {
	Delay     time.Duration
	Collected []kclient.Object
	Client    *Client
}

func (r *Response) RetryAfter(delay time.Duration) {
	if r.Delay == 0 || delay < r.Delay {
		r.Delay = delay
	}
}

func (r *Response) Objects(obj ...kclient.Object) {
	r.Collected = append(r.Collected, obj...)
}

func (c Client) Delete(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteOption) error {
	//TODO implement me
	panic("implement me")
}

func (c Client) Update(ctx context.Context, obj kclient.Object, opts ...kclient.UpdateOption) error {
	//TODO implement me
	panic("implement me")
}

func (c Client) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.PatchOption) error {
	//TODO implement me
	panic("implement me")
}

func (c Client) DeleteAllOf(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteAllOfOption) error {
	//TODO implement me
	panic("implement me")
}

func (c Client) Status() kclient.StatusWriter {
	//TODO implement me
	panic("implement me")
}

func (c Client) Scheme() *runtime.Scheme {
	return c.SchemeObj
}

func (c Client) RESTMapper() meta2.RESTMapper {
	//TODO implement me
	panic("implement me")
}
