package restconfig

import (
	"os"

	"github.com/rancher/wrangler/pkg/ratelimit"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func Default() (*rest.Config, error) {
	return New(scheme.Scheme)
}

func SetScheme(cfg *rest.Config, scheme *runtime.Scheme) *rest.Config {
	cfg.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	cfg.UserAgent = rest.DefaultKubernetesUserAgent()
	return cfg
}

func New(scheme *runtime.Scheme) (*rest.Config, error) {
	cfg, err := config.GetConfigWithContext(os.Getenv("CONTEXT"))
	if err != nil {
		return nil, err
	}
	cfg.RateLimiter = ratelimit.None
	return SetScheme(cfg, scheme), nil
}
