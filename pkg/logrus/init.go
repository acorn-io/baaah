package logrus

import (
	"github.com/acorn-io/baaah/pkg/log"
	"github.com/bombsimon/logrusr/v4"
	"github.com/sirupsen/logrus"
	crlog "sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	crlog.SetLogger(logrusr.New(logrus.StandardLogger()))
	log.SetLogger(logrus.StandardLogger())
}
