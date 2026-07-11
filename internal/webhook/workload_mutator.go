package webhook

import (
	"context"
	"net/http"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/injection"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type WorkloadMutator struct {
	Client  client.Client
	Decoder admission.Decoder
}

func (m *WorkloadMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	obj := &unstructured.Unstructured{}
	if err := m.Decoder.Decode(req, obj); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	ann := podTemplateAnnotations(obj)
	if !injection.WantsInjection(ann) {
		return admission.Allowed("injection disabled")
	}
	providers, err := injection.ResolveProviders(ctx, m.Client, req.Namespace, ann)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if len(providers) == 0 {
		return admission.Allowed("no matching resource providers found")
	}
	if err := injection.PatchWorkload(obj, providers); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	marshaled, err := obj.MarshalJSON()
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

func podTemplateAnnotations(obj *unstructured.Unstructured) map[string]string {
	ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
	if ann == nil {
		ann = obj.GetAnnotations()
	}
	if ann == nil {
		ann = map[string]string{}
	}
	return ann
}
