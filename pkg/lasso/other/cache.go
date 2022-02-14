package other

import (
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

type Indexer[T meta.Object] func(obj T) ([]string, error)

type Cache[T any, TP PObject[T]] interface {
	Get(namespace, name string) (TP, error)
	List(namespace string, selector labels.Selector) ([]TP, error)

	AddIndexer(indexName string, indexer Indexer[TP])
	GetByIndex(indexName, key string) ([]TP, error)
}

type objCache[T any, TP PObject[T]] struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *objCache[T, TP]) Get(namespace, name string) (TP, error) {
	obj, exists, err := c.indexer.GetByKey(namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(TP), nil
}

func (c *objCache[T, TP]) List(namespace string, selector labels.Selector) (ret []TP, err error) {
	err = cache.ListAllByNamespace(c.indexer, namespace, selector, func(m interface{}) {
		ret = append(ret, m.(TP))
	})
	return ret, err
}

func (c *objCache[T, TP]) AddIndexer(indexName string, indexer Indexer[TP]) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(TP))
		},
	}))
}

func (c *objCache[T, TP]) GetByIndex(indexName, key string) (result []TP, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]TP, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(TP))
	}
	return result, nil
}
