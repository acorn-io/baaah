package runtime

import (
	"context"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type SharedControllerHandler interface {
	OnChange(key string, obj runtime.Object) (runtime.Object, error)
}

type SharedController interface {
	Controller

	RegisterHandler(ctx context.Context, name string, handler SharedControllerHandler) error
}

type SharedControllerHandlerFunc func(key string, obj runtime.Object) (runtime.Object, error)

func (s SharedControllerHandlerFunc) OnChange(key string, obj runtime.Object) (runtime.Object, error) {
	return s(key, obj)
}

type sharedController struct {
	// this allows one to create a sharedcontroller but it will not actually be started
	// unless some aspect of the controllers informer is accessed or needed to be used
	deferredController func() (Controller, error)
	controller         Controller
	handler            *SharedHandler
	startLock          sync.Mutex
	started            bool
	startError         error
	client             kclient.Client
	gvk                schema.GroupVersionKind
}

func (s *sharedController) Cache() (cache.Cache, error) {
	return s.initController().Cache()
}

func (s *sharedController) Enqueue(namespace, name string) {
	s.initController().Enqueue(namespace, name)
}

func (s *sharedController) EnqueueAfter(namespace, name string, delay time.Duration) {
	s.initController().EnqueueAfter(namespace, name, delay)
}

func (s *sharedController) EnqueueKey(key string) {
	s.initController().EnqueueKey(key)
}

func (s *sharedController) initController() Controller {
	s.startLock.Lock()
	defer s.startLock.Unlock()

	if s.controller != nil {
		return s.controller
	}

	controller, err := s.deferredController()
	if err != nil {
		controller = newErrorController(err)
	}

	s.startError = err
	s.controller = controller
	return s.controller
}

func (s *sharedController) Start(ctx context.Context, workers int) error {
	s.startLock.Lock()
	defer s.startLock.Unlock()

	if s.startError != nil || s.controller == nil {
		return s.startError
	}

	if s.started {
		return nil
	}

	if err := s.controller.Start(ctx, workers); err != nil {
		return err
	}
	s.started = true

	context.AfterFunc(ctx, func() {
		s.startLock.Lock()
		defer s.startLock.Unlock()
		s.started = false
	})

	return nil
}

func (s *sharedController) RegisterHandler(ctx context.Context, name string, handler SharedControllerHandler) (returnErr error) {
	// Ensure that controller is initialized
	c := s.initController()

	getHandlerTransaction(ctx).do(func() {
		ctx, cancel := context.WithCancel(ctx)
		s.handler.Register(ctx, name, handler)

		defer func() {
			if returnErr == nil {
				context.AfterFunc(ctx, cancel)
			} else {
				cancel()
			}
		}()

		s.startLock.Lock()
		defer s.startLock.Unlock()
		if s.started {
			var (
				objList runtime.Object
				cache   cache.Cache
			)

			objList, returnErr = s.client.Scheme().New(schema.GroupVersionKind{
				Group:   s.gvk.Group,
				Version: s.gvk.Version,
				Kind:    s.gvk.Kind + "List",
			})
			cache, returnErr = s.controller.Cache()
			if returnErr != nil {
				return
			}
			if returnErr = cache.List(context.TODO(), objList.(kclient.ObjectList)); returnErr != nil {
				return
			}
			returnErr = meta.EachListItem(objList, func(obj runtime.Object) error {
				mObj := obj.(kclient.Object)
				c.EnqueueKey(keyFunc(mObj.GetNamespace(), mObj.GetName()))
				return nil
			})
		}
	})

	return nil
}
