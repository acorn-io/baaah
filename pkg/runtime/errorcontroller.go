package runtime

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/cache"
)

type errorController struct {
	err error
}

func newErrorController(err error) *errorController {
	return &errorController{err: err}
}

func (n *errorController) Enqueue(namespace, name string) {
}

func (n *errorController) EnqueueAfter(namespace, name string, delay time.Duration) {
}

func (n *errorController) EnqueueKey(key string) {
}

func (n *errorController) Cache() (cache.Cache, error) {
	return nil, n.err
}

func (n *errorController) Start(ctx context.Context, workers int) error {
	return nil
}
