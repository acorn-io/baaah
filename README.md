# baaah
`baaah` is a controller framework born out of frustration. It strives to provide a simple interface for watching and updating Kubernetes objects.

## Usage
The easiest way to get started with `baaah` is to create a default router and start registering handlers (or handler functions).

```go
package main

import (
	"context"

	"github.com/acorn-io/baaah"
	"github.com/acorn-io/baaah/pkg/router"
	"k8s.io/client-go/kubernetes/scheme"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	r, err := baaah.DefaultRouter("my-router", scheme.Scheme)
	if err != nil {
		panic(err)
	}

	r.Type(new(appsv1.Deployment)).HandlerFunc(handleDeployment)

	ctx := context.Background()
	err = r.Start(ctx)
	if err != nil {
		panic(err)
	}
	
	<-ctx.Done()
	// Background context was canceled. Do whatever cleanup necessary.
}

func handleDeployment(req router.Request, resp router.Response) error {
	// Act on the deployment, which can be retrieved via req.Object
	// If you need to create new objects based on the deployment, use resp.Objects()
	// A req.Client and req.Ctx are provided for instances when you need to directly interact with the Kubernetes API.
	deployment := req.Object.(*appsv1.Deployment)
	
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: deployment.Name + "-key",
			Namespace: deployment.Namespace,
		},
		Data: map[string][]byte{
			"token": []byte("secret-key"),
		},
	}
	
	resp.Objects(secret)
	return nil
}
```

When using the `req.Client` and `resp.Objects` in your handlers, there are some niceties built-in to be aware of.

When any object passed to `resp.Objects` changes, there is a "trigger" created under the hood that will ensure the object that created object goes through its handlers. For exampale,  if the implementation of the deployment handle above created a secret for the deployment by passing the secret to `resp.Objects`, then everytime that secret changed, the deployment handler would be called with the deployment that created that secret. This is to ensure that the secret always has the correct/expected values.

Similarly, anytime the `req.Client` is used, `baaah` will also create triggers for those objects. Meaning that if the deployment handler above used the client to list all secrets, then the deployment handlers would be triggered anytime any secret is created/updated. If the client was used to get a specific service, then the deployment handler would be triggered any time that specific service was changed. This is something to be mindful of to ensure you don't overload the Kubernetes API. Using selectors when listing or using get would be preferred.

