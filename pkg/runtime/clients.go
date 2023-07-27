package runtime

import (
	"time"

	"github.com/acorn-io/baaah/pkg/runtime/multi"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Runtime struct {
	Backend *Backend
}

type Config struct {
	Rest      *rest.Config
	Scheme    *runtime.Scheme
	Namespace string
}

func NewRuntime(cfg *rest.Config, scheme *runtime.Scheme) (*Runtime, error) {
	return NewRuntimeWithConfig(Config{
		Rest:   cfg,
		Scheme: scheme,
	})
}

func NewRuntimeForNamespace(cfg *rest.Config, namespace string, scheme *runtime.Scheme) (*Runtime, error) {
	return NewRuntimeWithConfigs(Config{Rest: cfg, Scheme: scheme, Namespace: namespace}, nil)
}

func NewRuntimeWithConfig(cfg Config) (*Runtime, error) {
	return NewRuntimeWithConfigs(cfg, nil)
}

func NewRuntimeWithConfigs(defaultConfig Config, cfgs map[string]Config) (*Runtime, error) {
	clients := make(map[string]client.Client, len(cfgs))
	cachedClients := make(map[string]client.Client, len(cfgs))
	caches := make(map[string]cache.Cache, len(cfgs))
	schemes := make(map[string]*runtime.Scheme, len(cfgs))

	for key, cfg := range cfgs {
		uncachedClient, cachedClient, theCache, err := getClients(cfg)
		if err != nil {
			return nil, err
		}

		clients[key] = uncachedClient
		caches[key] = theCache
		schemes[key] = cfg.Scheme
		cachedClients[key] = cachedClient
	}

	uncachedClient, cachedClient, theCache, err := getClients(defaultConfig)
	if err != nil {
		return nil, err
	}

	aggUncachedClient := multi.NewClient(uncachedClient, clients)
	aggCachedClient := multi.NewClient(cachedClient, cachedClients)
	aggCache := multi.NewCache(theCache, defaultConfig.Scheme, caches, schemes)

	factory := NewSharedControllerFactory(aggUncachedClient, aggCache, &SharedControllerFactoryOptions{
		// In baaah this is only invoked when a key fails to process
		DefaultRateLimiter: workqueue.NewMaxOfRateLimiter(
			// This will go .5, 1, 2, 4, 8 seconds, etc up until 15 minutes
			workqueue.NewItemExponentialFailureRateLimiter(500*time.Millisecond, 15*time.Minute),
		),
	})

	return &Runtime{
		Backend: newBackend(factory, newCacheClient(aggUncachedClient, aggCachedClient), aggCache),
	}, nil
}

func getClients(cfg Config) (uncachedClient client.WithWatch, cachedClient client.Client, theCache cache.Cache, err error) {
	uncachedClient, err = client.NewWithWatch(cfg.Rest, client.Options{
		Scheme: cfg.Scheme,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	var namespaces []string
	if cfg.Namespace != "" {
		namespaces = append(namespaces, cfg.Namespace)
	}

	theCache, err = cache.New(cfg.Rest, cache.Options{
		Scheme:     cfg.Scheme,
		Namespaces: namespaces,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	cachedClient, err = client.New(cfg.Rest, client.Options{
		Scheme: cfg.Scheme,
		Cache: &client.CacheOptions{
			Reader: theCache,
		},
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return uncachedClient, cachedClient, theCache, nil
}
