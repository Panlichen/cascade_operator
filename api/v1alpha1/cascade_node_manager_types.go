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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// CascadeShardLayout is the detail layout and other configurate for a shard
type CascadeShardLayout struct {
	// minNodes is minNodes for a shard
	MinNodes int64 `json:"minNodes"`
	// maxNodes is maxNodes for a shard
	MaxNodes int64 `json:"maxNodes"`
	// reservedNodeId is reservedNodeId for a shard
	ReservedNodeId []int64 `json:"reservedNodeId"`
	// deliveryMode is deliveryMode for a shard
	DeliveryMode string `json:"deliveryMode"`
	// profile is profile for a shard
	Profile string `json:"profile"`
	// assignedNodeId is assignedNodeId for a shard, omitted in spec, maintained in status
	AssignedNodeId []int64 `json:"assignedNodeId,omitempty"`
}

// CascadeSubgroupLayout wraps configuration for each shard in a subgroup
type CascadeSubgroupLayout struct {
	ShardLayout []CascadeShardLayout `json:"shardLayout"`
}

// CascadeType wraps configuration for each subgroup for a type
type CascadeType struct {
	// typeAlias is the name for a type
	TypeAlias      string                  `json:"typeAlias"`
	SubgroupLayout []CascadeSubgroupLayout `json:"layout"`
}

// CascadeNodeManagerSpec defines the desired state of CascadeNodeManager
type CascadeNodeManagerSpec struct {
	TypesSpec []CascadeType `json:"typesSpec"`
}

type CascadeNodeManagerStatus struct {
	TypesStatus []CascadeType `json:"typesStatus"`

	// maxReservedNodeId is the max reserved node id calculated from configMap
	MaxReservedNodeId int64 `json:"maxReservedNodeId"`

	// nextNodeIdToAssign is the next non-reserved node id to assign
	NextNodeIdToAssign int64 `json:"nextNodeIdToAssign"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CascadeNodeManager stores the node assingment policy defined by user and maintains
// current node status
type CascadeNodeManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CascadeNodeManagerSpec   `json:"spec"`
	Status CascadeNodeManagerStatus `json:"status"`
}

//+kubebuilder:object:root=true

// CascadeNodeManagerList contains a list of CascadeNodeManager
type CascadeNodeManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CascadeNodeManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cascade{}, &CascadeList{})
}
