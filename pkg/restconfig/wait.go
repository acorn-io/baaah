package restconfig

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

func WaitFor(ctx context.Context, cfg *rest.Config) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	start := time.Now()
	log := logrus.WithField("server", cfg.Host)
	log.Info("Waiting for Kubernetes API to be ready")

	cli, err := rest.UnversionedRESTClientFor(cfg)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			// Return close error along with the last ready error, if any
			return errors.Join(fmt.Errorf("context closed before %q was ready: %w", cfg.Host, ctx.Err()), err)
		default:
		}

		resp := cli.Get().AbsPath("/readyz").Do(ctx)
		log = log.WithField("elapsed", time.Since(start))
		if err = resp.Error(); err == nil {
			log.Info("Kubernetes API ready")
			break
		}

		log.WithError(err).Debug("Kubernetes API not ready")
		time.Sleep(2 * time.Second)
	}
	return nil
}
