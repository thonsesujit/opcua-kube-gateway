package main

import (
	"flag"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	v1alpha1 "github.com/opcua-kube-gateway/opcua-kube-gateway/api/v1alpha1"
	"github.com/opcua-kube-gateway/opcua-kube-gateway/internal/controller"
	"github.com/opcua-kube-gateway/opcua-kube-gateway/internal/exporter"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		os.Exit(1)
	}
	defer logger.Sync()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "opcua-kube-gateway.opcua.gateway.io",
	})
	if err != nil {
		logger.Fatal("unable to create manager", zap.Error(err))
	}

	promExporter := exporter.NewPrometheusExporter(logger, prometheus.DefaultRegisterer)

	reconciler := controller.NewOPCUASubscriptionReconciler(
		mgr.GetClient(),
		logger,
		mgr.GetEventRecorderFor("opcua-kube-gateway"),
		promExporter,
	)

	if err := reconciler.SetupWithManager(mgr); err != nil {
		logger.Fatal("unable to create controller", zap.Error(err))
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Fatal("unable to set up health check", zap.Error(err))
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Fatal("unable to set up ready check", zap.Error(err))
	}

	logger.Info("starting operator")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Fatal("operator exited with error", zap.Error(err))
	}
}
