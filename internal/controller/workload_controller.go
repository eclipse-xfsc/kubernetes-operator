package controller

import (
	"context"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/config"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/index"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/logging"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/metrics"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/registry"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkloadReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Config    *config.OperatorConfig
	Inventory *index.Inventory
	Registry  *registry.Registry
	Log       logging.Logger
}

func (r *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var d appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &d); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	metrics.ReconcileTotal.WithLabelValues("workload").Inc()
	requested := modules.RequestedTypes(&d)
	for _, m := range r.Registry.Modules() {
		ps, err := m.Provide(ctx, &d)
		if err != nil {
			return ctrl.Result{}, err
		}
		for _, p := range ps {
			r.Inventory.UpsertProvider(p)
		}
		cs, err := m.Consume(ctx, &d)
		if err != nil {
			return ctrl.Result{}, err
		}
		for _, c := range cs {
			r.Inventory.UpsertConsumer(c)
			res, err := m.Inject(ctx, modules.InjectionRequest{ConsumerNamespace: c.Namespace, ConsumerName: c.Name, ConsumerKind: c.Kind, RequestedTypes: c.RequestedTypes, Mode: d.Annotations[modules.InjectModeAnno]})
			if err != nil {
				return ctrl.Result{}, err
			}
			r.Inventory.UpsertInjection(*res)
			r.Inventory.UpsertAccount(modules.Account{Name: "xsfc-" + c.Name + "-" + c.Type, Namespace: c.Namespace, Type: c.Type, ConsumerNamespace: c.Namespace, ConsumerName: c.Name, CreatedBy: "xfsc-operator"})
		}
	}
	if len(requested) > 0 {
		r.Inventory.UpsertManifest(index.Manifest{APIVersion: "apps/v1", Kind: "Deployment", Name: d.Name, Namespace: d.Namespace, RequestedTypes: requested, Annotations: d.Annotations, Labels: d.Labels})
	}
	r.Log.Info("reconciled workload", logging.String("namespace", d.Namespace), logging.String("name", d.Name))
	return ctrl.Result{}, nil
}
func (r *WorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&appsv1.Deployment{}).Complete(r)
}
