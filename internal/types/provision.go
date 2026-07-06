package types

import "sigs.k8s.io/controller-runtime/pkg/client"

type ProvisionRequest struct {
	Type     string
	Provider Provider
	Consumer Consumer

	Client client.Client

	Config map[string]string
}

type ProvisionResult struct {
	Accounts []Account

	SecretName string
	SecretData map[string][]byte

	ConfigMapName string
	ConfigData    map[string]string

	Resources []client.Object
}
