package router

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ResponseWrapper struct {
	NoPrune     bool
	NoPruneGVKs []schema.GroupVersionKind
	Delay       time.Duration
	Objs        []kclient.Object
}

func (r *ResponseWrapper) DisablePrune() {
	r.NoPrune = true
}

func (r *ResponseWrapper) WithoutPruneGVKs(gvks ...schema.GroupVersionKind) {
	r.NoPruneGVKs = append(r.NoPruneGVKs, gvks...)
}

func (r *ResponseWrapper) RetryAfter(delay time.Duration) {
	r.Delay = delay
}

func (r *ResponseWrapper) Objects(obj ...kclient.Object) {
	r.Objs = append(r.Objs, obj...)
}
