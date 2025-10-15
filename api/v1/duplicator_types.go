/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type NamespaceSelector struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

type TargetResource struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// DuplicatorSpec defines the desired state of Duplicator.
type DuplicatorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// NamespaceSelector selects the namespaces to duplicate resources from.
	NamespaceSelector NamespaceSelector `json:"namespaceSelector,omitempty"`

	// TargetResources specifies the resources to duplicate.
	TargetResources []TargetResource `json:"targetResources,omitempty"`
}

// DuplicatorStatus defines the observed state of Duplicator.
type DuplicatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Duplicator is the Schema for the duplicators API.
type Duplicator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DuplicatorSpec   `json:"spec,omitempty"`
	Status DuplicatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DuplicatorList contains a list of Duplicator.
type DuplicatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Duplicator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Duplicator{}, &DuplicatorList{})
}
