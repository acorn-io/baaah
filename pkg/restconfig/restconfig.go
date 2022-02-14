package restconfig

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func New(scheme *runtime.Scheme) (*rest.Config, error) {
	cfg, err := config.GetConfigWithContext("")
	if err != nil {
		return nil, err
	}
	cfg.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	cfg.UserAgent = rest.DefaultKubernetesUserAgent()
	return cfg, err
}
