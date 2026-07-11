package controller

import (
	"context"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/injection"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InventoryLogger struct{ Client client.Client }

func (i *InventoryLogger) Start(ctx context.Context) error {
	log := ctrl.Log.WithName("inventory")
	var providers resourcesv1alpha1.ResourceProviderList
	if err := i.Client.List(ctx, &providers); err != nil {
		log.Error(err, "failed to list installed resource providers")
	} else {
		log.Info("installed resource providers", "count", len(providers.Items))
		for idx := range providers.Items {
			p := &providers.Items[idx]
			log.Info("resource provider installed", "name", p.Name, "namespace", p.Namespace, "type", p.Spec.Type, "staticEnvCount", len(p.Spec.Outputs.Env), "externalSecretCount", len(p.Spec.Outputs.ExternalSecrets))
		}
	}
	var deployments appsv1.DeploymentList
	if err := i.Client.List(ctx, &deployments); err != nil {
		log.Error(err, "failed to list deployments")
	} else {
		consumerCount := 0
		for idx := range deployments.Items {
			d := &deployments.Items[idx]
			ann := workloadAnnotations(d)
			if !injection.WantsInjection(ann) {
				continue
			}
			consumerCount++
			log.Info("consumer installed", "name", d.Name, "namespace", d.Namespace, "needs", injection.SplitCSV(ann[injection.AnnotationNeeds]), "providers", injection.SplitCSV(ann[injection.AnnotationProviders]))
		}
		log.Info("installed consumers", "count", consumerCount, "deploymentCount", len(deployments.Items))
	}
	<-ctx.Done()
	return nil
}

func (i *InventoryLogger) NeedLeaderElection() bool { return true }
