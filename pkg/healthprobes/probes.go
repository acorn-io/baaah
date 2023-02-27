package healthprobes

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/server/healthz"
)

type HealthCheckObj interface {
	GetReadyzProbePort() int
}

func StartReadyzPingHandler(obj HealthCheckObj) {
	port := obj.GetReadyzProbePort()
	if port != 0 {
		url := fmt.Sprintf("http://127.0.0.1:%d", port)
		resp, err := http.Get(url)
		if resp != nil {
			statusCode := resp.StatusCode
			_ = resp.Body.Close()
			// If readyz is already being handled locally, don't try to start another one
			if err == nil && statusCode != http.StatusOK {
				return
			}
		}

		watchDog := healthz.PingHealthz
		healthMux := http.NewServeMux()
		healthz.InstallReadyzHandler(healthMux, watchDog)
		go func() {
			logrus.Infof("Starting readyz handler at %d", port)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), healthMux); err != nil {
				logrus.Fatalf("failed to listen & server readyz http server from port %d: %v", port, err)
			}
		}()
	}
}
