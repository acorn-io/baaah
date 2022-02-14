package typed

import (
	"testing"

	"github.com/ibuildthecloud/baaah/pkg/router"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestGetList(t *testing.T) {
	req := &Request[*v1.Pod]{}
	if false {
		// just testing that this compiles
		_, _ = Get[*v1.Pod](req.Client, "foo", nil)
		_, _ = List[*v1.PodList](req.Client, nil)
	}
}

func TestFor(t *testing.T) {
	obj, h := Handler(func(req Request[*v1.Pod], resp router.Response) error {
		return nil
	})
	h.Handle(router.Request{}, nil)

	gvks, _, err := scheme.Scheme.ObjectKinds(obj)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, gvks, 1)
}
