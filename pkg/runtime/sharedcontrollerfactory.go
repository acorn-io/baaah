package runtime

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type SharedControllerFactory interface {
	ForKind(gvk schema.GroupVersionKind) (SharedController, error)
	Start(ctx context.Context, workers int) error
}

type SharedControllerFactoryOptions struct {
	DefaultRateLimiter workqueue.RateLimiter
	DefaultWorkers     int

	KindRateLimiter map[schema.GroupVersionKind]workqueue.RateLimiter
	KindWorkers     map[schema.GroupVersionKind]int
}

type sharedControllerFactory struct {
	controllerLock sync.RWMutex

	cache       cache.Cache
	client      kclient.Client
	controllers map[schema.GroupVersionKind]*sharedController

	rateLimiter     workqueue.RateLimiter
	workers         int
	kindRateLimiter map[schema.GroupVersionKind]workqueue.RateLimiter
	kindWorkers     map[schema.GroupVersionKind]int
}

func NewSharedControllerFactory(c kclient.Client, cache cache.Cache, opts *SharedControllerFactoryOptions) SharedControllerFactory {
	opts = applyDefaultSharedOptions(opts)
	return &sharedControllerFactory{
		cache:           cache,
		client:          c,
		controllers:     map[schema.GroupVersionKind]*sharedController{},
		workers:         opts.DefaultWorkers,
		kindWorkers:     opts.KindWorkers,
		rateLimiter:     opts.DefaultRateLimiter,
		kindRateLimiter: opts.KindRateLimiter,
	}
}

func applyDefaultSharedOptions(opts *SharedControllerFactoryOptions) *SharedControllerFactoryOptions {
	var newOpts SharedControllerFactoryOptions
	if opts != nil {
		newOpts = *opts
	}
	if newOpts.DefaultWorkers == 0 {
		newOpts.DefaultWorkers = 5
	}
	return &newOpts
}

func (s *sharedControllerFactory) Start(ctx context.Context, defaultWorkers int) error {
	s.controllerLock.Lock()
	defer s.controllerLock.Unlock()

	go func() {
		if err := s.cache.Start(ctx); err != nil {
			panic(err)
		}
	}()

	// copy so we can release the lock during cache wait
	controllersCopy := map[schema.GroupVersionKind]*sharedController{}
	for k, v := range s.controllers {
		controllersCopy[k] = v
	}

	// Do not hold lock while waiting because this can cause a deadlock if
	// one of the handlers you are waiting on tries to acquire this lock (by looking up
	// shared controller)
	s.controllerLock.Unlock()
	s.cache.WaitForCacheSync(ctx)
	s.controllerLock.Lock()

	for gvk, controller := range controllersCopy {
		w, err := s.getWorkers(gvk, defaultWorkers)
		if err != nil {
			return err
		}
		if err := controller.Start(ctx, w); err != nil {
			return err
		}
	}

	return nil
}

func (s *sharedControllerFactory) ForKind(gvk schema.GroupVersionKind) (SharedController, error) {
	controllerResult := s.byGVK(gvk)
	if controllerResult != nil {
		return controllerResult, nil
	}

	s.controllerLock.Lock()
	defer s.controllerLock.Unlock()

	controllerResult = s.controllers[gvk]
	if controllerResult != nil {
		return controllerResult, nil
	}

	handler := &SharedHandler{}

	controllerResult = &sharedController{
		deferredController: func() (Controller, error) {
			rateLimiter, ok := s.kindRateLimiter[gvk]
			if !ok {
				rateLimiter = s.rateLimiter
			}

			return New(gvk, s.client.Scheme(), s.cache, handler, &Options{
				RateLimiter: rateLimiter,
			})
		},
		handler: handler,
		client:  s.client,
	}

	s.controllers[gvk] = controllerResult
	return controllerResult, nil
}

func (s *sharedControllerFactory) getWorkers(gvk schema.GroupVersionKind, workers int) (int, error) {
	w, ok := s.kindWorkers[gvk]
	if ok {
		return w, nil
	}
	if workers > 0 {
		return workers, nil
	}
	return s.workers, nil
}

func (s *sharedControllerFactory) byGVK(gvk schema.GroupVersionKind) *sharedController {
	s.controllerLock.RLock()
	defer s.controllerLock.RUnlock()
	return s.controllers[gvk]
}
