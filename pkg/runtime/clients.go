package runtime

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Runtime struct {
	Backend *Backend
}

func NewRuntime(cfg *rest.Config, scheme *runtime.Scheme) (*Runtime, error) {
	return NewRuntimeForNamespace(cfg, "", scheme)
}

func NewRuntimeForNamespace(cfg *rest.Config, namespace string, scheme *runtime.Scheme) (*Runtime, error) {
	uncachedClient, err := client.NewWithWatch(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	var namespaces []string
	if namespace != "" {
		namespaces = append(namespaces, namespace)
	}

	cache, err := cache.New(cfg, cache.Options{
		Scheme:     scheme,
		Namespaces: namespaces,
	})
	if err != nil {
		return nil, err
	}
	factory := NewSharedControllerFactory(uncachedClient, cache, &SharedControllerFactoryOptions{
		// In baaah this is only invoked when a key fails to process
		DefaultRateLimiter: workqueue.NewMaxOfRateLimiter(
			// This will go .5, 1, 2, 4, 8 seconds, etc up until 15 minutes
			workqueue.NewItemExponentialFailureRateLimiter(500*time.Millisecond, 15*time.Minute),
		),
	})
	if err != nil {
		return nil, err
	}

	cachedClient, err := client.New(cfg, client.Options{
		Scheme: scheme,
		Cache: &client.CacheOptions{
			Reader: cache,
		},
	})
	if err != nil {
		return nil, err
	}

	return &Runtime{
		Backend: newBackend(factory, newCacheClient(uncachedClient, cachedClient), cache),
	}, nil
}
