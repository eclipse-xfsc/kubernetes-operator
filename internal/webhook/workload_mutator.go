package webhook

import (
	"context"
	"net/http"
	"time"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/injection"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type WorkloadMutator struct {
	Client  client.Client
	Decoder admission.Decoder
}

func (m *WorkloadMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	started := time.Now()
	log := ctrl.LoggerFrom(ctx).WithValues(
		"component", "admission-webhook",
		"operation", req.Operation,
		"kind", req.Kind.Kind,
		"namespace", req.Namespace,
		"name", req.Name,
		"requestUID", req.UID,
	)
	log.Info("admission request received")

	obj := &unstructured.Unstructured{}
	if err := m.Decoder.Decode(req, obj); err != nil {
		log.Error(err, "failed to decode admission request")
		return admission.Errored(http.StatusBadRequest, err)
	}

	ann := podTemplateAnnotations(obj)
	if !injection.WantsInjection(ann) {
		log.V(1).Info("admission allowed without mutation", "reason", "injection disabled")
		return admission.Allowed("injection disabled")
	}

	log.Info("injection requested", "needs", ann[injection.AnnotationNeeds])
	providers, err := injection.ResolveProviders(ctx, m.Client, req.Namespace, ann)
	if err != nil {
		log.Error(err, "failed to resolve resource providers")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if len(providers) == 0 {
		log.Info("admission allowed without mutation", "reason", "no matching resource providers")
		return admission.Allowed("no matching resource providers found")
	}

	providerNames := make([]string, 0, len(providers))
	for i := range providers {
		providerNames = append(providerNames, providers[i].Name)
	}

	if _, err := injection.PatchWorkload(obj, providers, nil, nil); err != nil {
		log.Error(err, "failed to mutate workload", "providers", providerNames)
		return admission.Errored(http.StatusInternalServerError, err)
	}
	marshaled, err := obj.MarshalJSON()
	if err != nil {
		log.Error(err, "failed to marshal mutated workload")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	log.Info("workload mutation prepared",
		"providers", providerNames,
		"duration", time.Since(started).String(),
	)
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
