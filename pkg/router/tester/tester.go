package tester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
	"github.com/rancher/wrangler/pkg/yaml"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Harness struct {
	Scheme         *runtime.Scheme
	Existing       []meta.Object
	ExpectedOutput []meta.Object
	ExpectedDelay  time.Duration
}

func genericToTyped(scheme *runtime.Scheme, objs []runtime.Object) ([]meta.Object, error) {
	result := make([]meta.Object, 0, len(objs))
	for _, obj := range objs {
		typedObj, err := scheme.New(obj.GetObjectKind().GroupVersionKind())
		if err != nil {
			return nil, err
		}

		bytes, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(bytes, typedObj); err != nil {
			return nil, err
		}
		result = append(result, typedObj.(meta.Object))
	}
	return result, nil
}

func readFile(scheme *runtime.Scheme, base, file string) ([]meta.Object, error) {
	path := filepath.Join(base, file)
	data, err := ioutil.ReadFile(filepath.Join(base, file))
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	objs, err := yaml.ToObjects(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("unmarshalling %s: %w", path, err)
	}
	return genericToTyped(scheme, objs)
}

func FromDir(scheme *runtime.Scheme, path string) (*Harness, meta.Object, error) {
	input, err := readFile(scheme, path, "input.yaml")
	if err != nil {
		return nil, nil, err
	}

	if len(input) != 1 {
		return nil, nil, fmt.Errorf("%s/%s does not include one input object", path, "input.yaml")
	}

	existing, err := readFile(scheme, path, "existing.yaml")
	if err != nil {
		return nil, nil, err
	}

	expected, err := readFile(scheme, path, "expected.yaml")
	if err != nil {
		return nil, nil, err
	}

	return &Harness{
		Scheme:         scheme,
		Existing:       existing,
		ExpectedOutput: expected,
		ExpectedDelay:  0,
	}, input[0], nil
}

func DefaultTest(t *testing.T, scheme *runtime.Scheme, path string, handler router.HandlerFunc) {
	t.Run(path, func(t *testing.T) {
		harness, input, err := FromDir(scheme, path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = harness.Invoke(t, input, handler)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func (b *Harness) InvokeFunc(t *testing.T, input meta.Object, handler router.HandlerFunc) (*Response, error) {
	return b.Invoke(t, input, handler)
}

func (b *Harness) Invoke(t *testing.T, input meta.Object, handler router.Handler) (*Response, error) {
	gvk, err := apiutil.GVKForObject(input, b.Scheme)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	deadline, ok := t.Deadline()
	if ok {
		newCtx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()
		ctx = newCtx
	}

	var (
		client = &Client{
			DefaultNamespace: input.GetNamespace(),
			Objects:          b.Existing,
			Scheme:           b.Scheme,
		}
		resp = Response{
			Client: client,
		}
		req = router.Request{
			Client:      client,
			Object:      input,
			Ctx:         ctx,
			GVK:         gvk,
			Namespace:   input.GetNamespace(),
			Name:        input.GetName(),
			Key:         toKey(input.GetNamespace(), input.GetName()),
			FromTrigger: false,
		}
	)

	err = handler.Handle(req, &resp)
	if err != nil {
		return &resp, err
	}

	expected, err := toObjectMap(b.Scheme, b.ExpectedOutput)
	if err != nil {
		return &resp, err
	}

	collected, err := toObjectMap(b.Scheme, resp.Collected)
	if err != nil {
		return &resp, err
	}

	assert.Equal(t, b.ExpectedDelay, resp.Delay)

	if len(b.ExpectedOutput) == 0 {
		return &resp, nil
	}

	expectedKeys := map[ObjectKey]bool{}
	for k := range expected {
		expectedKeys[k] = true
	}
	collectedKeys := map[ObjectKey]bool{}
	for k := range collected {
		collectedKeys[k] = true
	}

	for key := range collectedKeys {
		if !expectedKeys[key] {
			t.Fatalf("Unexpected object %s/%s: %v", key.Namespace, key.Name, key.GVK)
		}
	}

	for key := range expectedKeys {
		if !collectedKeys[key] {
			t.Fatalf("Missing expected object %s/%s: %v", key.Namespace, key.Name, key.GVK)
		}
		assert.Equal(t, expected[key], collected[key], "object %s/%s (%v) does not match", key.Namespace, key.Name, key.GVK)
	}

	return &resp, nil
}

func toObjectMap(scheme *runtime.Scheme, objs []meta.Object) (map[ObjectKey]meta.Object, error) {
	result := map[ObjectKey]meta.Object{}
	for _, o := range objs {
		gvk, err := apiutil.GVKForObject(o, scheme)
		if err != nil {
			return nil, err
		}
		o.GetObjectKind().SetGroupVersionKind(gvk)
		result[ObjectKey{
			GVK:       gvk,
			Namespace: o.GetNamespace(),
			Name:      o.GetName(),
		}] = o
	}
	return result, nil
}

type ObjectKey struct {
	GVK       schema.GroupVersionKind
	Namespace string
	Name      string
}

func toKey(ns, name string) string {
	if ns == "" {
		return name
	}
	return ns + "/" + name
}
