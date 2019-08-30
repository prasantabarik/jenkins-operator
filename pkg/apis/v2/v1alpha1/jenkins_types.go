package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JenkinsSpec defines the desired state of Jenkins
// +k8s:openapi-gen=true
type JenkinsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Image   string           `json:"image"`
	Version string           `json:"version"`
	Volumes []JenkinsVolumes `json:"volumes, omitempty"`
}

type JenkinsVolumes struct {
	Name         string `json:"name"`
	StorageClass string `json:"storage_class"`
	Capacity     string `json:"capacity"`
}

// JenkinsStatus defines the observed state of Jenkins
// +k8s:openapi-gen=true
type JenkinsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Available       bool      `json:"available, omitempty"`
	LastTimeUpdated time.Time `json:"lastTimeUpdated, omitempty"`
	Status          string    `json:"status, omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Jenkins is the Schema for the jenkins API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Jenkins struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JenkinsSpec   `json:"spec,omitempty"`
	Status JenkinsStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsList contains a list of Jenkins
type JenkinsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jenkins `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Jenkins{}, &JenkinsList{})
}