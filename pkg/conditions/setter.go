package conditions

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/acorn-io/baaah/pkg/router"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Conditions interface {
	GetConditions() *[]metav1.Condition
}

// ErrTerminal is an error that can not be recovered from until some other state in the
// system changes, typically from additional user input.
type ErrTerminal struct {
	Cause error
}

func NewErrTerminal(cause error) *ErrTerminal {
	return &ErrTerminal{
		Cause: cause,
	}
}

func NewErrTerminalf(format string, args ...any) *ErrTerminal {
	return &ErrTerminal{
		Cause: fmt.Errorf(format, args...),
	}
}

func (e *ErrTerminal) Error() string {
	return e.Cause.Error()
}

func (e *ErrTerminal) Unwrap() error {
	return e.Cause
}

func ErrorMiddleware() router.Middleware {
	return func(h router.Handler) router.Handler {
		return router.HandlerFunc(func(req router.Request, resp router.Response) error {
			var (
				uErr   *ErrTerminal
				logErr error
			)

			t, ok := req.Object.(Conditions)
			if !ok {
				return h.Handle(req, resp)
			}

			err := h.Handle(req, resp)
			if errors.As(err, &uErr) {
				logErr = uErr
			} else if apierrors.IsNotFound(err) {
				logErr = err
			}

			existing, ok := resp.Attributes()["_errormiddleware"].(*metav1.Condition)
			if !ok {
				existing = meta.FindStatusCondition(*t.GetConditions(), "Controller")
				if existing != nil {
					cp := *existing
					resp.Attributes()["_errormiddleware"] = &cp
				}
			}

			errored, _ := resp.Attributes()["_errormiddleware:errored"].(bool)
			if errored {
				if logErr != nil {
					return nil
				}
				return err
			}

			if logErr != nil {
				slog.Error("error processing controller",
					"err", logErr,
					"gvk", req.GVK,
					"namespace", req.Namespace,
					"name", req.Name)
				if existing != nil {
					meta.SetStatusCondition(t.GetConditions(), *existing)
				}
				meta.SetStatusCondition(t.GetConditions(), metav1.Condition{
					Type:               "Controller",
					Status:             metav1.ConditionFalse,
					ObservedGeneration: req.Object.GetGeneration(),
					Reason:             reflect.TypeOf(logErr).Name(),
					Message:            logErr.Error(),
				})
				resp.Attributes()["_errormiddleware:errored"] = true
				resp.DisablePrune()
				return nil
			}

			if err == nil {
				if existing != nil {
					meta.SetStatusCondition(t.GetConditions(), *existing)
				}
				meta.SetStatusCondition(t.GetConditions(), metav1.Condition{
					Type:               "Controller",
					Status:             metav1.ConditionTrue,
					ObservedGeneration: req.Object.GetGeneration(),
				})
				return nil
			} else {
				return err
			}
		})
	}
}
