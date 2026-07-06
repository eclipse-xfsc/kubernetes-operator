package webhook

import (
	"context"
	"net/http"
	"strings"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
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
	ann := obj.GetAnnotations()
	if ann["inject.xfsc.io/enabled"] != "true" {
		return admission.Allowed("injection disabled")
	}

	wanted := split(ann["inject.xfsc.io/types"])
	var list resourcesv1alpha1.ResourceBindingList
	if err := m.Client.List(ctx, &list, client.InNamespace(req.Namespace)); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	bindings := []resourcesv1alpha1.ResourceBinding{}
	providers := map[string]resourcesv1alpha1.ResourceProvider{}
	for _, b := range list.Items {
		if b.Spec.ConsumerRef.Name != req.Name || b.Spec.ConsumerRef.Kind != req.Kind.Kind {
			continue
		}
		if len(wanted) > 0 && !contains(wanted, b.Spec.Type) {
			continue
		}
		bindings = append(bindings, b)
		pns := b.Spec.ProviderRef.Namespace
		if pns == "" {
			pns = req.Namespace
		}
		var p resourcesv1alpha1.ResourceProvider
		if err := m.Client.Get(ctx, client.ObjectKey{Name: b.Spec.ProviderRef.Name, Namespace: pns}, &p); err == nil {
			providers[pns+"/"+p.Name] = p
		}
	}
	if len(bindings) == 0 {
		return admission.Allowed("no resource bindings found")
	}
	if err := injection.PatchWorkload(obj, bindings, providers); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	marshaled, err := obj.MarshalJSON()
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

func split(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
