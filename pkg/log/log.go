package log

import (
	"log"
)

var (
	// Stupid log abstraction.  It's expected that the consumer
	// of this library override these methods like
	//
	//  log.Infof = logrus.Infof
	//  log.Errorf = logrus.Errorf
	//
	//  or you can call SetLogger

	Infof = func(message string, obj ...interface{}) {
		//log.Printf("INFO: "+message+"\n", obj...)
	}
	Warnf = func(message string, obj ...interface{}) {
		log.Printf("WARN [BAAAH]: "+message+"\n", obj...)
	}
	Errorf = func(message string, obj ...interface{}) {
		log.Printf("ERROR[BAAAH]: "+message+"\n", obj...)
	}
	Fatalf = func(message string, obj ...interface{}) {
		log.Fatalf("FATAL[BAAAH]: "+message+"\n", obj...)
	}
	Debugf = func(message string, obj ...interface{}) {
		//log.Printf("DEBUG: "+message+"\n", obj...)
	}
)

type Logger interface {
	Infof(message string, obj ...interface{})
	Warnf(message string, obj ...interface{})
	Errorf(message string, obj ...interface{})
	Fatalf(message string, obj ...interface{})
	Debugf(message string, obj ...interface{})
}

func SetLogger(logger Logger) {
	Debugf = logger.Debugf
	Infof = logger.Infof
	Warnf = logger.Warnf
	Errorf = logger.Errorf
	Fatalf = logger.Fatalf
}
