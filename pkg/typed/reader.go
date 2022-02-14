package typed

import (
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
)

func Get[T meta.Object](client router.Reader, name string, opts *meta.GetOptions) (T, error) {
	obj := New[T]()
	err := client.Get(obj, name, opts)
	return obj, err
}

func List[T meta.ObjectList](client router.Reader, opts *meta.ListOptions) (T, error) {
	obj := New[T]()
	err := client.List(obj, opts)
	return obj, err
}
