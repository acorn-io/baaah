package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/acorn-io/baaah/pkg/log"
	"gomodules.xyz/jsonpatch/v2"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	jsonPatchType = v1.PatchTypeJSONPatch
)

func NewRouter() *Router {
	return &Router{}
}

type Router struct {
	matches []*RouteMatch
}

func (r *Router) sendError(rw http.ResponseWriter, review *v1.AdmissionReview, err error) {
	log.Errorf("%v", err)
	if review == nil || review.Request == nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	review.Response.Allowed = false
	review.Response.Result = &errors.NewInternalError(err).ErrStatus
	writeResponse(rw, review)
}

func writeResponse(rw http.ResponseWriter, review *v1.AdmissionReview) {
	rw.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(rw).Encode(review)
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	review := &v1.AdmissionReview{}
	err := json.NewDecoder(req.Body).Decode(review)
	if err != nil {
		r.sendError(rw, review, err)
		return
	}

	if review.Request == nil {
		r.sendError(rw, review, fmt.Errorf("request is not set"))
		return
	}

	response := &Response{
		AdmissionResponse: v1.AdmissionResponse{
			UID: review.Request.UID,
		},
	}

	review.Response = &response.AdmissionResponse

	if err := r.admit(response, review.Request, req); err != nil {
		r.sendError(rw, review, err)
		return
	}

	writeResponse(rw, review)
}

func (r *Router) admit(response *Response, request *v1.AdmissionRequest, req *http.Request) error {
	for _, m := range r.matches {
		if m.matches(request) {
			err := m.admit(response, &Request{
				AdmissionRequest: *request,
				Context:          req.Context(),
			})
			log.Debugf("admit result: %s %s %s user=%s allowed=%v err=%v", request.Operation, request.Kind.String(), resourceString(request.Namespace, request.Name), request.UserInfo.Username, response.Allowed, err)
			return err
		}
	}
	return fmt.Errorf("no route match found for %s %s %s", request.Operation, request.Kind.String(), resourceString(request.Namespace, request.Name))
}

func (r *Router) next() *RouteMatch {
	match := &RouteMatch{}
	r.matches = append(r.matches, match)
	return match
}

type Request struct {
	v1.AdmissionRequest

	Context context.Context
}

func (r *Request) DecodeOldObject(obj kclient.Object) error {
	return json.Unmarshal(r.OldObject.Raw, obj)
}

func (r *Request) DecodeObject(obj kclient.Object) error {
	return json.Unmarshal(r.Object.Raw, obj)
}

type Response struct {
	v1.AdmissionResponse
}

func (r *Response) CreatePatch(request *Request, newObj kclient.Object) error {
	if len(r.Patch) > 0 {
		return fmt.Errorf("response patch has already been already been assigned")
	}

	newBytes, err := json.Marshal(newObj)
	if err != nil {
		return err
	}

	patch, err := jsonpatch.CreatePatch(request.Object.Raw, newBytes)
	if err != nil {
		return err
	}

	patchData, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	r.Patch = patchData
	r.PatchType = &jsonPatchType
	return nil
}

type Handler interface {
	Admit(resp *Response, req *Request) error
}

type HandlerFunc func(resp *Response, req *Request) error

func (h HandlerFunc) Admit(resp *Response, req *Request) error {
	return h(resp, req)
}

// resourceString returns the resource formatted as a string
func resourceString(ns, name string) string {
	if ns == "" {
		return name
	}
	return fmt.Sprintf("%s/%s", ns, name)
}
