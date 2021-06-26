/*
Copyright 2021.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CascadeConfigMap is where the controller gets layout info for a Cascade instance
type CascadeConfigMap struct {
	// name indicates the name of ConfigMap provided by the user
	Name string `json:"name"`
	// jsonItem indicates the name of the layout json file item inside the ConfigMap
	JsonItem string `json:"jsonItem"`
}

// CascadeSpec defines the desired state of Cascade
type CascadeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster

	ConfigMap CascadeConfigMap `json:"configMap"`

	// logicalSize is the desired logical number of a Cascade Group
	LogicalSize int64 `json:"logicalSize"`
}

// CascadeStatus defines the observed state of Cascade
type CascadeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Nodes are the names of the cascade pods
	Nodes []string `json:"nodes"`

	// realSize is the physical number of nodes.
	RealSize int64 `json:"realSize"`
	// logicalSize is the logical number of nodes, which means that overlapped nodes
	// are counted for each appearance
	LogicalSize int64 `json:"logicalSize"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Cascade is the Schema for the cascades API
type Cascade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CascadeSpec   `json:"spec,omitempty"`
	Status CascadeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CascadeList contains a list of Cascade
type CascadeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cascade `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cascade{}, &CascadeList{})
}
