package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/acorn-io/baaah/pkg/log"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgocache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const maxTimeout2min = 2 * time.Minute

type Handler interface {
	OnChange(key string, obj runtime.Object) error
}

type ResourceVersionGetter interface {
	GetResourceVersion() string
}

type HandlerFunc func(key string, obj runtime.Object) error

func (h HandlerFunc) OnChange(key string, obj runtime.Object) error {
	return h(key, obj)
}

type Controller interface {
	Enqueue(namespace, name string)
	EnqueueAfter(namespace, name string, delay time.Duration)
	EnqueueKey(key string)
	Informer() clientgocache.SharedIndexInformer
	Start(ctx context.Context, workers int) error
}

type controller struct {
	startLock sync.Mutex

	name         string
	workqueue    workqueue.RateLimitingInterface
	rateLimiter  workqueue.RateLimiter
	informer     clientgocache.SharedIndexInformer
	handler      Handler
	gvk          schema.GroupVersionKind
	startKeys    []startKey
	started      bool
	registration clientgocache.ResourceEventHandlerRegistration
	obj          runtime.Object
	cache        cache.Cache
}

type startKey struct {
	key   string
	after time.Duration
}

type Options struct {
	RateLimiter workqueue.RateLimiter
}

func New(gvk schema.GroupVersionKind, scheme *runtime.Scheme, cache cache.Cache, handler Handler, opts *Options) (Controller, error) {
	opts = applyDefaultOptions(opts)

	obj, err := newObject(scheme, gvk)
	if err != nil {
		return nil, err
	}

	controller := &controller{
		gvk:         gvk,
		name:        gvk.String(),
		handler:     handler,
		cache:       cache,
		obj:         obj,
		rateLimiter: opts.RateLimiter,
	}

	return controller, nil
}

func newObject(scheme *runtime.Scheme, gvk schema.GroupVersionKind) (runtime.Object, error) {
	obj, err := scheme.New(gvk)
	if runtime.IsNotRegisteredError(err) {
		return &unstructured.Unstructured{}, nil
	}
	return obj, err
}

func applyDefaultOptions(opts *Options) *Options {
	var newOpts Options
	if opts != nil {
		newOpts = *opts
	}
	if newOpts.RateLimiter == nil {
		newOpts.RateLimiter = workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemFastSlowRateLimiter(time.Millisecond, maxTimeout2min, 30),
			workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 30*time.Second),
		)
	}
	return &newOpts
}

func (c *controller) Informer() clientgocache.SharedIndexInformer {
	return c.informer
}

func (c *controller) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *controller) run(ctx context.Context, workers int) {
	defer func() {
		_ = c.informer.RemoveEventHandler(c.registration)
	}()

	c.startLock.Lock()
	// we have to defer queue creation until we have a stopCh available because a workqueue
	// will create a goroutine under the hood.  It we instantiate a workqueue we must have
	// a mechanism to Shutdown it down.  Without the stopCh we don't know when to shutdown
	// the queue and release the goroutine
	c.workqueue = workqueue.NewNamedRateLimitingQueue(c.rateLimiter, c.name)
	for _, start := range c.startKeys {
		if start.after == 0 {
			c.workqueue.Add(start.key)
		} else {
			c.workqueue.AddAfter(start.key, start.after)
		}
	}
	c.startKeys = nil
	c.startLock.Unlock()

	defer utilruntime.HandleCrash()
	defer func() {
		c.workqueue.ShutDown()
	}()

	// Start the informer factories to begin populating the informer caches
	log.Infof("Starting %s controller", c.name)

	for i := 0; i < workers; i++ {
		go wait.Until(func() {
			c.runWorker(ctx)
		}, time.Second, ctx.Done())
	}

	<-ctx.Done()
	c.startLock.Lock()
	defer c.startLock.Unlock()
	c.started = false
	log.Infof("Shutting down %s workers", c.name)
}

func (c *controller) Start(ctx context.Context, workers int) error {
	c.startLock.Lock()
	defer c.startLock.Unlock()

	if c.started {
		return nil
	}

	if c.informer == nil {
		informer, err := c.cache.GetInformerForKind(ctx, c.gvk)
		if err != nil {
			return err
		}
		if sii, ok := informer.(clientgocache.SharedIndexInformer); ok {
			c.informer = sii
		} else {
			return fmt.Errorf("expecting cache.SharedIndexInformer but got %T", informer)
		}
	}

	if c.registration == nil {
		registration, err := c.informer.AddEventHandler(clientgocache.ResourceEventHandlerFuncs{
			AddFunc: c.handleObject,
			UpdateFunc: func(old, new interface{}) {
				c.handleObject(new)
			},
			DeleteFunc: c.handleObject,
		})
		if err != nil {
			return err
		}
		c.registration = registration
	}

	if !c.informer.HasSynced() {
		go func() {
			_ = c.cache.Start(ctx)
		}()
	}

	if ok := clientgocache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	go c.run(ctx, workers)
	c.started = true
	return nil
}

func (c *controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *controller) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	if err := c.processSingleItem(ctx, obj); err != nil {
		if !strings.Contains(err.Error(), "please apply your changes to the latest version and try again") {
			log.Errorf("%v", err)
		}
		return true
	}

	return true
}

func (c *controller) processSingleItem(ctx context.Context, obj interface{}) error {
	var (
		key string
		ok  bool
	)

	defer c.workqueue.Done(obj)

	if key, ok = obj.(string); !ok {
		c.workqueue.Forget(obj)
		log.Errorf("expected string in workqueue but got %#v", obj)
		return nil
	}
	if err := c.syncHandler(ctx, key); err != nil {
		c.workqueue.AddRateLimited(key)
		return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
	}

	c.workqueue.Forget(obj)
	return nil
}

func (c *controller) syncHandler(ctx context.Context, key string) error {
	ns, name := keyParse(key)
	obj := c.obj.DeepCopyObject().(kclient.Object)
	err := c.cache.Get(ctx, kclient.ObjectKey{
		Name:      name,
		Namespace: ns,
	}, obj)
	if apierror.IsNotFound(err) {
		return c.handler.OnChange(key, nil)
	} else if err != nil {
		return err
	}

	return c.handler.OnChange(key, obj.(runtime.Object))
}

func (c *controller) EnqueueKey(key string) {
	c.startLock.Lock()
	defer c.startLock.Unlock()

	if c.workqueue == nil {
		c.startKeys = append(c.startKeys, startKey{key: key})
	} else {
		c.workqueue.Add(key)
	}
}

func (c *controller) Enqueue(namespace, name string) {
	key := keyFunc(namespace, name)

	c.startLock.Lock()
	defer c.startLock.Unlock()

	if c.workqueue == nil {
		c.startKeys = append(c.startKeys, startKey{key: key})
	} else {
		c.workqueue.AddRateLimited(key)
	}
}

func (c *controller) EnqueueAfter(namespace, name string, duration time.Duration) {
	key := keyFunc(namespace, name)

	c.startLock.Lock()
	defer c.startLock.Unlock()

	if c.workqueue == nil {
		c.startKeys = append(c.startKeys, startKey{key: key, after: duration})
	} else {
		c.workqueue.AddAfter(key, duration)
	}
}

func keyParse(key string) (namespace string, name string) {
	var ok bool
	namespace, name, ok = strings.Cut(key, "/")
	if !ok {
		name = namespace
		namespace = ""
	}
	return
}

func keyFunc(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}

func (c *controller) enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = clientgocache.MetaNamespaceKeyFunc(obj); err != nil {
		log.Errorf("%v", err)
		return
	}
	c.startLock.Lock()
	if c.workqueue == nil {
		c.startKeys = append(c.startKeys, startKey{key: key})
	} else {
		c.workqueue.Add(key)
	}
	c.startLock.Unlock()
}

func (c *controller) handleObject(obj interface{}) {
	if _, ok := obj.(metav1.Object); !ok {
		tombstone, ok := obj.(clientgocache.DeletedFinalStateUnknown)
		if !ok {
			log.Errorf("error decoding object, invalid type")
			return
		}
		newObj, ok := tombstone.Obj.(metav1.Object)
		if !ok {
			log.Errorf("error decoding object tombstone, invalid type")
			return
		}
		obj = newObj
	}
	c.enqueue(obj)
}
