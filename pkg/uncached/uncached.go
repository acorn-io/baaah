package uncached

import (
	"k8s.io/apimachinery/pkg/runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func List(obj kclient.ObjectList) kclient.ObjectList {
	return &HolderList{
		ObjectList: obj,
	}
}

func Get(obj kclient.Object) kclient.Object {
	return &Holder{
		Object: obj,
	}
}

type Holder struct {
	kclient.Object
}

func (h *Holder) DeepCopyObject() runtime.Object {
	return &Holder{Object: h.Object.DeepCopyObject().(kclient.Object)}
}

type HolderList struct {
	kclient.ObjectList
}

func (h *HolderList) DeepCopyObject() runtime.Object {
	return &HolderList{ObjectList: h.ObjectList.DeepCopyObject().(kclient.ObjectList)}
}
