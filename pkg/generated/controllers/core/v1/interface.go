package v1

import (
	"github.com/ibuildthecloud/baaah/pkg/lasso"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/schemes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	PodResource = "pods"
	PodKind     = "Pod"
	Group       = ""
	Version     = "v1"
)

var (
	PodGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: PodKind}
)

func init() {
	schemes.Register(v1.AddToScheme)
}

type PodController lasso.Controller[v1.Pod]
type PodClient lasso.Client[v1.Pod]

func NewPodClient(client *client.Client) PodClient {
	return lasso.NewClient[v1.Pod](client)
}

func NewPodController(controllerFactory controller.SharedControllerFactory) PodController {
	return lasso.NewController[v1.Pod](PodGVK, PodResource, true, controllerFactory)
}
