package runtime

import (
	"encoding/json"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func CopyInto(dst, src runtime.Object) error {
	if _, ok := src.(*unstructured.Unstructured); ok {
		data, err := json.Marshal(src)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, dst)
	}
	src = src.DeepCopyObject()
	dstVal := reflect.ValueOf(dst)
	srcVal := reflect.ValueOf(src)
	if !srcVal.Type().AssignableTo(dstVal.Type()) {
		return fmt.Errorf("type %s not assignable to %s", srcVal.Type(), dstVal.Type())
	}
	reflect.Indirect(dstVal).Set(reflect.Indirect(srcVal))
	return nil
}
