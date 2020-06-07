package v1alpha1

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	GroupName    = "toolchain.openshift.dev"
	GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}
)

// ToolchainSpec defines the desired state of Toolchain
// +k8s:openapi-gen=true
type ToolchainSpec struct {
	Components []runtime.RawExtension `json:"components" protobuf:"bytes,3,rep,name=components"`
}

// ToolchainStatus defines the observed state of Toolchain
// +k8s:openapi-gen=true
type ToolchainStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Last known condition of the OpenShift Pipelines operator installation.
	// Supported condition types:
	// ToolchainReady
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Conditions"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:io.kubernetes.conditions"
	Conditions []toolchainv1alpha1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Toolchain defines which operators should be installed and how
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=toolchains,scope=Cluster
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"ToolchainReady\")].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type==\"ToolchainReady\")].reason"
// +kubebuilder:validation:XPreserveUnknownFields
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="CodeReady Toolchain"
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
type Toolchain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToolchainSpec   `json:"spec,omitempty"`
	Status ToolchainStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ToolchainList contains a list of Toolchain
type ToolchainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Toolchain `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Toolchain{}, &ToolchainList{})
}
