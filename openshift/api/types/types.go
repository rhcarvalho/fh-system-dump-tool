// Package types contains simplified type definitions from Kubernetes and
// OpenShift. References:
// https://github.com/openshift/origin/blob/v1.3.0-rc1/vendor/k8s.io/kubernetes/pkg/api/v1/types.go
// https://github.com/openshift/origin/blob/v1.3.0-rc1/pkg/deploy/api/v1/types.go
package types

// TypeMeta describes an individual object in an API response or request
// with strings representing the type of the object.
type TypeMeta struct {
	Kind string `json:"kind,omitempty"`
}

// ObjectMeta is metadata that all persisted resources must have.
type ObjectMeta struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// ObjectReference contains enough information to let you inspect or modify the referred object.
type ObjectReference struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// PodList is a list of Pods.
type PodList struct {
	Items []Pod `json:"items"`
}

// Pod is a collection of containers that can run on a host.
type Pod struct {
	ObjectMeta `json:"metadata,omitempty"`
	Status     PodStatus `json:"status,omitempty"`
}

// EventList is a list of events.
type EventList struct {
	Items []Event `json:"items"`
}

// Event is a report of an event somewhere in the cluster.
type Event struct {
	TypeMeta       `json:",inline"`
	InvolvedObject ObjectReference `json:"involvedObject"`
	Reason         string          `json:"reason,omitempty"`
	Message        string          `json:"message,omitempty"`
	Count          int32           `json:"count,omitempty"`
	Type           string          `json:"type,omitempty"`
}

// PodStatus represents information about the status of a pod.
type PodStatus struct {
	ContainerStatuses []ContainerStatus `json:"containerStatuses,omitempty"`
}

// ContainerStatus contains details for the current status of this container.
type ContainerStatus struct {
	Name  string         `json:"name"`
	State ContainerState `json:"state,omitempty"`
}

// ContainerState holds a possible state of container.
type ContainerState struct {
	Waiting *ContainerStateWaiting `json:"waiting,omitempty"`
}

// ContainerStateWaiting is a waiting state of a container.
type ContainerStateWaiting struct {
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

// DeploymentConfigList is a collection of deployment configs.
type DeploymentConfigList struct {
	Items []DeploymentConfig `json:"items"`
}

// DeploymentConfig represents a configuration for a single deployment.
type DeploymentConfig struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       DeploymentConfigSpec `json:"spec"`
}

// DeploymentConfigSpec represents the desired state of the deployment.
type DeploymentConfigSpec struct {
	Replicas int32 `json:"replicas"`
}
