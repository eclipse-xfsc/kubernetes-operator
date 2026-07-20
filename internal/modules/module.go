package modules

import (
	"context"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Request struct {
	Client      client.Client
	Provider    resourcesv1alpha1.ResourceProvider
	Claim       *resourcesv1alpha1.ResourceClaim
	Namespace   string
	Workload    string
	Annotations map[string]string
}

type Result struct {
	Resources  []*unstructured.Unstructured
	SecretData map[string][]byte
}

type Module interface {
	Type() string
	Reconcile(context.Context, Request) (Result, error)
}
