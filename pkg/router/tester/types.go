package tester

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/google/uuid"
	"github.com/rancher/wrangler/pkg/randomtoken"
	"k8s.io/apimachinery/pkg/api/errors"
	meta2 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Client struct {
	DefaultNamespace string
	Objects          []meta.Object
	Scheme           *runtime.Scheme
	Created          []meta.Object
}

func (c Client) objects() []meta.Object {
	return append(c.Objects, c.Created...)
}

func (c *Client) Get(out meta.Object, name string, opts *meta.GetOptions) error {
	t := reflect.TypeOf(out)
	ns := c.DefaultNamespace
	if opts != nil && opts.Namespace != "" {
		ns = opts.Namespace
	}
	for _, obj := range c.objects() {
		if reflect.TypeOf(obj) != t {
			continue
		}
		if obj.GetName() == name &&
			obj.GetNamespace() == ns {
			copy(out, obj)
			return nil
		}
	}
	return errors.NewNotFound(schema.GroupResource{
		Group:    fmt.Sprintf("Unknown group from test: %T", out),
		Resource: fmt.Sprintf("Unknown resource from test: %T", out),
	}, name)
}

func copy(dest, src meta.Object) {
	srcCopy := src.DeepCopyObject()
	reflect.Indirect(reflect.ValueOf(dest)).Set(reflect.Indirect(reflect.ValueOf(srcCopy)))
}

func (c *Client) List(objList meta.ObjectList, opts *meta.ListOptions) error {
	gvk, err := apiutil.GVKForObject(objList, c.Scheme)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(gvk.Kind, "List") {
		return fmt.Errorf("invalid list object %v, Kind must end with List", gvk)
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	genericObj, err := c.Scheme.New(gvk)
	if err != nil {
		return err
	}
	obj := genericObj.(meta.Object)
	t := reflect.TypeOf(obj)
	ns := c.DefaultNamespace
	if opts != nil && opts.Namespace != "" {
		ns = opts.Namespace
	}
	var resultObjs []runtime.Object
	for _, testObj := range c.objects() {
		if testObj.GetNamespace() != ns {
			continue
		}
		if reflect.TypeOf(obj) != t {
			continue
		}
		if opts != nil && !opts.Selector.Matches(labels.Set(testObj.GetLabels())) {
			continue
		}
		copy(obj, testObj)
		if err != nil {
			return err
		}

		resultObjs = append(resultObjs, testObj)
		newObj, err := c.Scheme.New(gvk)
		if err != nil {
			return err
		}
		obj = newObj.(meta.Object)
	}
	return meta2.SetList(objList, resultObjs)
}

func (c *Client) Delete(obj meta.Object) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Update(obj meta.Object) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) UpdateStatus(obj meta.Object) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Create(obj meta.Object) error {
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
	Collected []meta.Object
	Client    *Client
}

func (r *Response) RetryAfter(delay time.Duration) {
	if r.Delay == 0 || delay < r.Delay {
		r.Delay = delay
	}
}

func (r *Response) Objects(obj ...meta.Object) {
	r.Collected = append(r.Collected, obj...)
}
