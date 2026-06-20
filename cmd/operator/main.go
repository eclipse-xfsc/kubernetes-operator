package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	apiadapter "github.com/eclipse-xfsc/kubernetes-operator/internal/api"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/config"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/controller"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/index"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/logging"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/metrics"
	postgresmodule "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/postgres"
	redismodule "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/redis"
	s3module "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/s3"
	telemetrymodule "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/telemetry"
	vaultmodule "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/vault"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/registry"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/runtimeinfo"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(resourcesv1alpha1.AddToScheme(scheme))
}
func main() {
	var metricsAddr, probeAddr, apiAddr, configPath string
	var leader bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "")
	flag.StringVar(&apiAddr, "api-bind-address", ":8088", "")
	flag.StringVar(&configPath, "config", "/etc/xsfc-resource-operator/config.yaml", "")
	flag.BoolVar(&leader, "leader-elect", false, "")
	flag.Parse()
	log := logging.New()
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	cfg, err := config.LoadOrDefault(configPath)
	if err != nil {
		log.Fatal("unable to load config", logging.Error(err))
	}
	inv := index.NewInventory()
	reg := registry.New()
	reg.MustRegister(telemetrymodule.New(), postgresmodule.New(), redismodule.New(), s3module.New(), vaultmodule.New())
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme, HealthProbeBindAddress: probeAddr, LeaderElection: leader, LeaderElectionID: "xsfc-resource-operator.xfsc.io"})
	if err != nil {
		log.Fatal("unable to create manager", logging.Error(err))
	}
	metrics.Register()
	if err := (&controller.WorkloadReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Config: cfg, Inventory: inv, Registry: reg, Log: log.Named("workload-controller")}).SetupWithManager(mgr); err != nil {
		log.Fatal("unable to create workload controller", logging.Error(err))
	}
	if err := (&controller.ResourceProfileReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Inventory: inv, Registry: reg, Log: log.Named("resourceprofile-controller")}).SetupWithManager(mgr); err != nil {
		log.Fatal("unable to create resourceprofile controller", logging.Error(err))
	}
	_ = mgr.AddHealthzCheck("healthz", healthz.Ping)
	_ = mgr.AddReadyzCheck("readyz", healthz.Ping)
	srv := apiadapter.NewServer(apiadapter.ServerConfig{Address: apiAddr, Version: runtimeinfo.Info{OperatorVersion: runtimeinfo.OperatorVersion, GitCommit: runtimeinfo.GitCommit, BuildDate: runtimeinfo.BuildDate}, Inventory: inv, Registry: reg, Logger: log.Named("api")})
	ctx, cancel := context.WithCancel(ctrl.SetupSignalHandler())
	defer cancel()
	go func() {
		log.Info("starting REST API", logging.String("address", apiAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("REST API failed", logging.Error(err))
		}
	}()
	log.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		log.Fatal("manager stopped", logging.Error(err))
		os.Exit(1)
	}
}
