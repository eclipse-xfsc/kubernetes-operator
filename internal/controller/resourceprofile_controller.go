package controller

import (
	"context"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/index"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/logging"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/metrics"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/registry"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceProfileReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Inventory *index.Inventory
	Registry  *registry.Registry
	Log       logging.Logger
}

func (r *ResourceProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var p resourcesv1alpha1.ResourceProfile
	if err := r.Get(ctx, req.NamespacedName, &p); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	metrics.ReconcileTotal.WithLabelValues("resourceprofile").Inc()
	for _, e := range p.Spec.Exports {
		r.Inventory.UpsertProvider(modules.Provider{Type: e.Type, Name: e.Name, Namespace: p.Namespace, Kind: "ResourceProfile", Resource: p.Name, Module: "resourceprofile"})
	}
	for _, n := range p.Spec.Requires {
		r.Inventory.UpsertConsumer(modules.Consumer{Type: n.Type, Name: p.Name, Namespace: p.Namespace, Kind: "ResourceProfile", Resource: p.Name, RequestedTypes: []string{n.Type}})
	}
	r.Log.Info("reconciled resource profile", logging.String("namespace", p.Namespace), logging.String("name", p.Name))
	return ctrl.Result{}, nil
}
func (r *ResourceProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&resourcesv1alpha1.ResourceProfile{}).Complete(r)
}
