package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PortAddressTranslation describes a port address translation.
type PortAddressTranslation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PortAddressTranslationSpec `json:"spec"`
}

// PortAddressTranslationSpec is the spec for a PortAddressTranslation resource
type PortAddressTranslationSpec struct {
	// REQUIRED: Name of the serice to map
	Service string `json:"service"`

	// REQUIRED: A valid non-negative integer port number.
	Port int `json:"port"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PortAddressTranslationList is a list of PortAddressTranslation resources
type PortAddressTranslationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PortAddressTranslation `json:"items"`
}
