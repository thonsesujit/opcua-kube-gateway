// Package v1alpha1 contains API Schema definitions for the opcua.gateway.io v1alpha1 API group.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecurityMode defines the OPC-UA security mode.
// +kubebuilder:validation:Enum=None;Sign;SignAndEncrypt
type SecurityMode string

const (
	SecurityModeNone           SecurityMode = "None"
	SecurityModeSign           SecurityMode = "Sign"
	SecurityModeSignAndEncrypt SecurityMode = "SignAndEncrypt"
)

// SubscriptionPhase describes the current lifecycle phase.
type SubscriptionPhase string

const (
	PhaseConnecting   SubscriptionPhase = "Connecting"
	PhaseConnected    SubscriptionPhase = "Connected"
	PhaseError        SubscriptionPhase = "Error"
	PhaseDisconnected SubscriptionPhase = "Disconnected"
)

// OPCUANode defines a single OPC-UA node to subscribe to.
type OPCUANode struct {
	// NodeID is the OPC-UA node identifier (e.g., "ns=2;s=Temperature").
	// +kubebuilder:validation:MinLength=1
	NodeID string `json:"nodeId"`

	// Name is the metric name for this node (lowercase, underscores only).
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9_]*$`
	Name string `json:"name"`

	// Unit is the unit of measurement label (optional).
	// +optional
	Unit string `json:"unit,omitempty"`

	// Interval is the subscription publishing interval (default "5s").
	// +optional
	// +kubebuilder:default="5s"
	Interval string `json:"interval,omitempty"`
}

// PrometheusExporterConfig configures the Prometheus metrics exporter.
type PrometheusExporterConfig struct {
	// Enabled controls whether Prometheus metrics are exported.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Prefix is the metric name prefix (default "opcua_").
	// +optional
	// +kubebuilder:default="opcua_"
	Prefix string `json:"prefix,omitempty"`
}

// ExporterConfig holds configuration for all exporters.
type ExporterConfig struct {
	// Prometheus configures the Prometheus metrics exporter.
	// +optional
	Prometheus PrometheusExporterConfig `json:"prometheus,omitempty"`
}

// OPCUASubscriptionSpec defines the desired state of an OPCUASubscription.
type OPCUASubscriptionSpec struct {
	// Endpoint is the OPC-UA server URL (e.g., "opc.tcp://plc:4840").
	// +kubebuilder:validation:MinLength=1
	Endpoint string `json:"endpoint"`

	// SecurityMode is the OPC-UA security mode (default "None").
	// +optional
	// +kubebuilder:default=None
	SecurityMode SecurityMode `json:"securityMode,omitempty"`

	// Nodes is the list of OPC-UA nodes to subscribe to.
	// +kubebuilder:validation:MinItems=1
	Nodes []OPCUANode `json:"nodes"`

	// Exporters configures how data is exported.
	// +optional
	Exporters ExporterConfig `json:"exporters,omitempty"`
}

// NodeStatus holds the status of a single OPC-UA node subscription.
type NodeStatus struct {
	// NodeID is the OPC-UA node identifier.
	NodeID string `json:"nodeId"`

	// LastValue is the last received value as a string.
	// +optional
	LastValue string `json:"lastValue,omitempty"`

	// LastUpdated is the timestamp of the last value update.
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// Error is the error message if the node subscription failed.
	// +optional
	Error string `json:"error,omitempty"`
}

// OPCUASubscriptionStatus defines the observed state of an OPCUASubscription.
type OPCUASubscriptionStatus struct {
	// Phase is the current lifecycle phase.
	// +optional
	Phase SubscriptionPhase `json:"phase,omitempty"`

	// LastConnected is the timestamp of the last successful connection.
	// +optional
	LastConnected *metav1.Time `json:"lastConnected,omitempty"`

	// Message is a human-readable status message.
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions represent the latest available observations of the resource's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Nodes holds per-node subscription status.
	// +optional
	Nodes []NodeStatus `json:"nodes,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.spec.endpoint`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OPCUASubscription is the Schema for the opcuasubscriptions API.
type OPCUASubscription struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OPCUASubscriptionSpec   `json:"spec,omitempty"`
	Status OPCUASubscriptionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OPCUASubscriptionList contains a list of OPCUASubscription.
type OPCUASubscriptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OPCUASubscription `json:"items"`
}
