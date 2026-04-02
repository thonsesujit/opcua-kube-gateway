// Package controller implements the Kubernetes controller for OPCUASubscription resources.
package controller

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1alpha1 "github.com/opcua-kube-gateway/opcua-kube-gateway/api/v1alpha1"
	"github.com/opcua-kube-gateway/opcua-kube-gateway/internal/exporter"
	opcuaclient "github.com/opcua-kube-gateway/opcua-kube-gateway/internal/opcua"
	"github.com/gopcua/opcua/ua"
)

const (
	finalizerName    = "opcua.gateway.io/finalizer"
	conditionReady   = "Ready"
	conditionConnected = "Connected"
)

// subscriptionState tracks the runtime state for a single OPCUASubscription.
type subscriptionState struct {
	cancelFn context.CancelFunc
	client   *opcuaclient.Client
}

// OPCUASubscriptionReconciler reconciles OPCUASubscription resources.
type OPCUASubscriptionReconciler struct {
	client.Client
	Logger   *zap.Logger
	Recorder record.EventRecorder
	Exporter *exporter.PrometheusExporter

	mu     sync.Mutex
	states map[string]*subscriptionState // key: namespace/name
}

// NewOPCUASubscriptionReconciler creates a new reconciler.
func NewOPCUASubscriptionReconciler(
	c client.Client,
	logger *zap.Logger,
	recorder record.EventRecorder,
	exp *exporter.PrometheusExporter,
) *OPCUASubscriptionReconciler {
	return &OPCUASubscriptionReconciler{
		Client:   c,
		Logger:   logger,
		Recorder: recorder,
		Exporter: exp,
		states:   make(map[string]*subscriptionState),
	}
}

// +kubebuilder:rbac:groups=opcua.gateway.io,resources=opcuasubscriptions,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=opcua.gateway.io,resources=opcuasubscriptions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=opcua.gateway.io,resources=opcuasubscriptions/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile handles create/update/delete for OPCUASubscription resources.
func (r *OPCUASubscriptionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.With(zap.String("subscription", req.NamespacedName.String()))

	var sub v1alpha1.OPCUASubscription
	if err := r.Get(ctx, req.NamespacedName, &sub); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	key := req.NamespacedName.String()

	// Handle deletion
	if !sub.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &sub, key, log)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&sub, finalizerName) {
		controllerutil.AddFinalizer(&sub, finalizerName)
		if err := r.Update(ctx, &sub); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile the subscription
	return r.reconcileSubscription(ctx, &sub, key, log)
}

func (r *OPCUASubscriptionReconciler) handleDeletion(
	ctx context.Context,
	sub *v1alpha1.OPCUASubscription,
	key string,
	log *zap.Logger,
) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(sub, finalizerName) {
		return ctrl.Result{}, nil
	}

	log.Info("handling deletion")

	r.stopSubscription(key, log)
	r.Exporter.UnregisterSubscription(sub.Namespace, sub.Name)
	r.Exporter.ConnectionsActive.Dec()
	r.Exporter.SubscriptionsActive.Dec()

	r.Recorder.Event(sub, corev1.EventTypeNormal, "Disconnected", "OPC-UA session closed due to deletion")

	controllerutil.RemoveFinalizer(sub, finalizerName)
	if err := r.Update(ctx, sub); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OPCUASubscriptionReconciler) reconcileSubscription(
	ctx context.Context,
	sub *v1alpha1.OPCUASubscription,
	key string,
	log *zap.Logger,
) (ctrl.Result, error) {
	// Register metrics for all nodes
	prefix := sub.Spec.Exporters.Prometheus.Prefix
	if prefix == "" {
		prefix = "opcua_"
	}

	subLabels := exporter.SubscriptionLabels{
		Namespace:    sub.Namespace,
		Subscription: sub.Name,
		Endpoint:     sub.Spec.Endpoint,
	}

	for _, node := range sub.Spec.Nodes {
		nodeLabels := exporter.NodeLabels{
			NodeID: node.NodeID,
			Unit:   node.Unit,
		}
		if err := r.Exporter.RegisterNode(prefix, node.Name, subLabels, nodeLabels); err != nil {
			log.Error("failed to register metric", zap.Error(err), zap.String("nodeID", node.NodeID))
		}
	}

	// Check if subscription is already running
	r.mu.Lock()
	existing := r.states[key]
	r.mu.Unlock()

	if existing != nil {
		// Already running — in MVP, restart on spec change
		r.stopSubscription(key, log)
	}

	// Build client config
	nodes := make([]opcuaclient.NodeConfig, len(sub.Spec.Nodes))
	for i, n := range sub.Spec.Nodes {
		interval, _ := time.ParseDuration(n.Interval)
		if interval <= 0 {
			interval = 5 * time.Second
		}
		nodes[i] = opcuaclient.NodeConfig{
			NodeID:   n.NodeID,
			Interval: interval,
		}
	}

	cfg := opcuaclient.ClientConfig{
		Endpoint:     sub.Spec.Endpoint,
		SecurityMode: securityModeToUA(sub.Spec.SecurityMode),
		Nodes:        nodes,
	}

	// Start connection in background
	subCtx, cancel := context.WithCancel(context.Background())
	opcClient := opcuaclient.NewClient(cfg, log, r.makeOnChange(sub.Namespace, sub.Name, sub.Spec.Nodes))

	r.mu.Lock()
	r.states[key] = &subscriptionState{
		cancelFn: cancel,
		client:   opcClient,
	}
	r.mu.Unlock()

	// Update status to Connecting
	r.updateStatus(ctx, sub, v1alpha1.PhaseConnecting, "Connecting to OPC-UA server", log)
	r.Recorder.Event(sub, corev1.EventTypeNormal, "Connecting", fmt.Sprintf("Connecting to %s", sub.Spec.Endpoint))

	go r.runSubscription(subCtx, sub, key, opcClient, log)

	return ctrl.Result{}, nil
}

func (r *OPCUASubscriptionReconciler) runSubscription(
	ctx context.Context,
	sub *v1alpha1.OPCUASubscription,
	key string,
	opcClient *opcuaclient.Client,
	log *zap.Logger,
) {
	r.Exporter.ConnectionsActive.Inc()
	r.Exporter.SubscriptionsActive.Inc()

	err := opcClient.Connect(ctx)
	if err != nil && ctx.Err() == nil {
		log.Error("subscription failed", zap.Error(err))
		r.Exporter.ErrorsTotal.WithLabelValues("connection").Inc()

		subCopy := sub.DeepCopy()
		r.updateStatus(context.Background(), subCopy, v1alpha1.PhaseError, err.Error(), log)
		r.Recorder.Event(sub, corev1.EventTypeWarning, "Error", err.Error())
	}
}

func (r *OPCUASubscriptionReconciler) makeOnChange(
	namespace, subscription string,
	nodes []v1alpha1.OPCUANode,
) func(opcuaclient.DataChange) {
	return func(dc opcuaclient.DataChange) {
		key := exporter.NodeMetricKey{
			Namespace:    namespace,
			Subscription: subscription,
			NodeID:       dc.NodeID,
		}

		val, ok := toFloat64(dc.Value)
		if ok {
			r.Exporter.UpdateNode(key, val)
		}
	}
}

func (r *OPCUASubscriptionReconciler) stopSubscription(key string, log *zap.Logger) {
	r.mu.Lock()
	state, exists := r.states[key]
	if exists {
		delete(r.states, key)
	}
	r.mu.Unlock()

	if state != nil {
		state.cancelFn()
		if err := state.client.Close(); err != nil {
			log.Warn("error closing OPC-UA client", zap.Error(err))
		}
	}
}

func (r *OPCUASubscriptionReconciler) updateStatus(
	ctx context.Context,
	sub *v1alpha1.OPCUASubscription,
	phase v1alpha1.SubscriptionPhase,
	message string,
	log *zap.Logger,
) {
	sub.Status.Phase = phase
	sub.Status.Message = message

	isReady := phase == v1alpha1.PhaseConnected
	condition := metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionFalse,
		Reason:             string(phase),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
	if isReady {
		condition.Status = metav1.ConditionTrue
		now := metav1.Now()
		sub.Status.LastConnected = &now
	}

	meta.SetStatusCondition(&sub.Status.Conditions, condition)

	if err := r.Status().Update(ctx, sub); err != nil {
		log.Warn("failed to update status", zap.Error(err))
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *OPCUASubscriptionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OPCUASubscription{}).
		Complete(r)
}

func securityModeToUA(mode v1alpha1.SecurityMode) ua.MessageSecurityMode {
	switch mode {
	case v1alpha1.SecurityModeSign:
		return ua.MessageSecurityModeSign
	case v1alpha1.SecurityModeSignAndEncrypt:
		return ua.MessageSecurityModeSignAndEncrypt
	default:
		return ua.MessageSecurityModeNone
	}
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	case bool:
		if val {
			return 1, true
		}
		return 0, true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
