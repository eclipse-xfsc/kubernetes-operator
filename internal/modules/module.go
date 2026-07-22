package modules

import (
	"context"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type ProvisionRequest struct {
	Provider    resourcesv1alpha1.ResourceProvider
	Claim       resourcesv1alpha1.ResourceClaim
	AdminSecret corev1.Secret
	ClaimSecret corev1.Secret
}

type Provisioner interface {
	Type() string
	Provision(context.Context, ProvisionRequest) error
}
