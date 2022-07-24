package router

import "k8s.io/utils/strings/slices"

type FinalizerHandler struct {
	FinalizerID string
	Next        Handler
}

func (f FinalizerHandler) Handle(req Request, resp Response) error {
	obj := req.Object
	if obj == nil {
		return nil
	}

	if obj.GetDeletionTimestamp().IsZero() {
		if !slices.Contains(obj.GetFinalizers(), f.FinalizerID) {
			obj.SetFinalizers(append(obj.GetFinalizers(), f.FinalizerID))
			if err := req.Client.Update(req.Ctx, obj); err != nil {
				return err
			}
			resp.Objects(obj)
		}
		return nil
	}

	if len(obj.GetFinalizers()) == 0 || obj.GetFinalizers()[0] != f.FinalizerID {
		return nil
	}

	return f.Next.Handle(req, resp)
}
