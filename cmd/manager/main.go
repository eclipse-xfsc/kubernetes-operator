package main

import (
	"crypto/tls"
	"flag"
	"os"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/config"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/controller"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules/cassandra"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules/nats"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules/postgres"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules/redis"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules/s3"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules/vault"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/webhook"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(resourcesv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr, probeAddr, moduleConfigPath string
	var webhookPort int
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "metrics address")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "probe address")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "leader election")
	flag.IntVar(&webhookPort, "webhook-port", 9443, "webhook server port")
	flag.StringVar(&moduleConfigPath, "module-config", os.Getenv("XFSC_MODULE_CONFIG"), "path to module configuration")
	zapOpts := zap.Options{Development: true}
	zapOpts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))
	setupLog := ctrl.Log.WithName("setup")
	setupLog.Info("starting resource operator", "leaderElection", enableLeaderElection, "metricsAddress", metricsAddr, "probeAddress", probeAddr)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "resource-operator.xfsc.io",
		WebhookServer:          ctrlwebhook.NewServer(ctrlwebhook.Options{Port: webhookPort, TLSOpts: []func(*tls.Config){}}),
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	if err := mgr.Add(&controller.InventoryLogger{Client: mgr.GetClient()}); err != nil {
		setupLog.Error(err, "unable to register inventory logger")
		os.Exit(1)
	}

	moduleConfig, err := config.LoadModules(moduleConfigPath)
	if err != nil {
		setupLog.Error(err, "unable to load module configuration")
		os.Exit(1)
	}

	provisioners := make([]modules.Provisioner, 0, 6)
	if moduleConfig.Redis.IsEnabled() {
		provisioners = append(provisioners, redis.New(redis.NewBackend()))
	}
	if moduleConfig.Postgres.IsEnabled() {
		provisioners = append(provisioners, postgres.New(postgres.NewBackend()))
	}
	if moduleConfig.Cassandra.IsEnabled() {
		provisioners = append(provisioners, cassandra.New(cassandra.NewBackend()))
	}
	if moduleConfig.NATS.IsEnabled() {
		provisioners = append(provisioners, nats.New(nats.NewBackend()))
	}
	if moduleConfig.S3.IsEnabled() {
		provisioners = append(provisioners, s3.New(s3.NewBackend()))
	}
	if moduleConfig.Vault.IsEnabled() {
		provisioners = append(provisioners, vault.New(nil))
	}
	moduleRegistry := modules.NewRegistry(provisioners...)

	if err := (&controller.ResourceClaimReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("resource-claim-controller"), Modules: moduleRegistry,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to register resource claim controller")
		os.Exit(1)
	}

	if err := (&controller.WorkloadReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("resource-operator"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to register workload controller")
		os.Exit(1)
	}
	mgr.GetWebhookServer().Register("/mutate-workloads", &ctrlwebhook.Admission{Handler: &webhook.WorkloadMutator{Client: mgr.GetClient(), Decoder: admission.NewDecoder(mgr.GetScheme())}})
	setupLog.Info("registered mutating webhook", "path", "/mutate-workloads", "port", webhookPort)
	_ = mgr.AddHealthzCheck("healthz", healthz.Ping)
	_ = mgr.AddReadyzCheck("readyz", healthz.Ping)
	setupLog.Info("manager starting")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "manager stopped with error")
		os.Exit(1)
	}
}
