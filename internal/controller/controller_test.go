package controller

import (
	"testing"

	v1alpha1 "github.com/opcua-kube-gateway/opcua-kube-gateway/api/v1alpha1"
	"github.com/gopcua/opcua/ua"
)

func TestSecurityModeToUA(t *testing.T) {
	tests := []struct {
		input    v1alpha1.SecurityMode
		expected ua.MessageSecurityMode
	}{
		{v1alpha1.SecurityModeNone, ua.MessageSecurityModeNone},
		{v1alpha1.SecurityModeSign, ua.MessageSecurityModeSign},
		{v1alpha1.SecurityModeSignAndEncrypt, ua.MessageSecurityModeSignAndEncrypt},
		{"", ua.MessageSecurityModeNone}, // default
	}

	for _, tt := range tests {
		result := securityModeToUA(tt.input)
		if result != tt.expected {
			t.Errorf("securityModeToUA(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		ok       bool
	}{
		{"float64", float64(42.5), 42.5, true},
		{"float32", float32(42.5), float64(float32(42.5)), true},
		{"int", 42, 42.0, true},
		{"int32", int32(42), 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"uint32", uint32(42), 42.0, true},
		{"uint64", uint64(42), 42.0, true},
		{"bool true", true, 1.0, true},
		{"bool false", false, 0.0, true},
		{"string number", "42.5", 42.5, true},
		{"string invalid", "not-a-number", 0, false},
		{"nil", nil, 0, false},
		{"slice", []int{1, 2}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat64(tt.input)
			if ok != tt.ok {
				t.Errorf("toFloat64(%v): ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if ok && result != tt.expected {
				t.Errorf("toFloat64(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
