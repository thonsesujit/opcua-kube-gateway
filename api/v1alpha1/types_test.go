package v1alpha1_test

import (
	"testing"

	v1alpha1 "github.com/opcua-kube-gateway/opcua-kube-gateway/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func validSubscription() v1alpha1.OPCUASubscription {
	return v1alpha1.OPCUASubscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sub",
			Namespace: "default",
		},
		Spec: v1alpha1.OPCUASubscriptionSpec{
			Endpoint:     "opc.tcp://plc:4840",
			SecurityMode: v1alpha1.SecurityModeNone,
			Nodes: []v1alpha1.OPCUANode{
				{
					NodeID:   "ns=2;s=Temperature",
					Name:     "pump_temperature",
					Unit:     "celsius",
					Interval: "5s",
				},
			},
			Exporters: v1alpha1.ExporterConfig{
				Prometheus: v1alpha1.PrometheusExporterConfig{
					Enabled: true,
					Prefix:  "opcua_",
				},
			},
		},
	}
}

func TestOPCUASubscription_ValidSpec(t *testing.T) {
	sub := validSubscription()

	if sub.Spec.Endpoint != "opc.tcp://plc:4840" {
		t.Errorf("expected endpoint opc.tcp://plc:4840, got %s", sub.Spec.Endpoint)
	}
	if sub.Spec.SecurityMode != v1alpha1.SecurityModeNone {
		t.Errorf("expected SecurityModeNone, got %s", sub.Spec.SecurityMode)
	}
	if len(sub.Spec.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(sub.Spec.Nodes))
	}
	if sub.Spec.Nodes[0].NodeID != "ns=2;s=Temperature" {
		t.Errorf("expected nodeId ns=2;s=Temperature, got %s", sub.Spec.Nodes[0].NodeID)
	}
	if sub.Spec.Nodes[0].Name != "pump_temperature" {
		t.Errorf("expected name pump_temperature, got %s", sub.Spec.Nodes[0].Name)
	}
}

func TestOPCUASubscription_DefaultValues(t *testing.T) {
	sub := v1alpha1.OPCUASubscription{
		Spec: v1alpha1.OPCUASubscriptionSpec{
			Endpoint: "opc.tcp://plc:4840",
			Nodes: []v1alpha1.OPCUANode{
				{NodeID: "ns=2;s=Temp", Name: "temp"},
			},
		},
	}

	// SecurityMode defaults (set by kubebuilder, tested here as struct default)
	if sub.Spec.SecurityMode != "" {
		t.Errorf("expected empty default for SecurityMode before defaulting, got %s", sub.Spec.SecurityMode)
	}
}

func TestOPCUASubscription_StatusPhases(t *testing.T) {
	phases := []v1alpha1.SubscriptionPhase{
		v1alpha1.PhaseConnecting,
		v1alpha1.PhaseConnected,
		v1alpha1.PhaseError,
		v1alpha1.PhaseDisconnected,
	}

	expected := []string{"Connecting", "Connected", "Error", "Disconnected"}
	for i, phase := range phases {
		if string(phase) != expected[i] {
			t.Errorf("phase %d: expected %s, got %s", i, expected[i], phase)
		}
	}
}

func TestOPCUASubscription_DeepCopy(t *testing.T) {
	original := validSubscription()
	original.Status = v1alpha1.OPCUASubscriptionStatus{
		Phase:   v1alpha1.PhaseConnected,
		Message: "connected",
		Nodes: []v1alpha1.NodeStatus{
			{NodeID: "ns=2;s=Temperature", LastValue: "42.5"},
		},
	}

	copied := original.DeepCopy()

	if copied.Spec.Endpoint != original.Spec.Endpoint {
		t.Error("deep copy did not preserve Endpoint")
	}
	if copied.Status.Phase != original.Status.Phase {
		t.Error("deep copy did not preserve Phase")
	}

	// Mutating the copy should not affect the original
	copied.Spec.Nodes[0].Name = "mutated"
	if original.Spec.Nodes[0].Name == "mutated" {
		t.Error("deep copy shares memory with original for Nodes slice")
	}

	copied.Status.Nodes[0].LastValue = "99.9"
	if original.Status.Nodes[0].LastValue == "99.9" {
		t.Error("deep copy shares memory with original for status Nodes slice")
	}
}

func TestOPCUASubscription_SecurityModeValues(t *testing.T) {
	modes := map[v1alpha1.SecurityMode]string{
		v1alpha1.SecurityModeNone:           "None",
		v1alpha1.SecurityModeSign:           "Sign",
		v1alpha1.SecurityModeSignAndEncrypt: "SignAndEncrypt",
	}

	for mode, expected := range modes {
		if string(mode) != expected {
			t.Errorf("expected %s, got %s", expected, mode)
		}
	}
}

func TestOPCUASubscriptionList_DeepCopy(t *testing.T) {
	list := v1alpha1.OPCUASubscriptionList{
		Items: []v1alpha1.OPCUASubscription{validSubscription()},
	}

	copied := list.DeepCopy()
	if len(copied.Items) != 1 {
		t.Fatalf("expected 1 item in copied list, got %d", len(copied.Items))
	}

	copied.Items[0].Spec.Endpoint = "mutated"
	if list.Items[0].Spec.Endpoint == "mutated" {
		t.Error("deep copy list shares memory with original")
	}
}
