/*
Copyright 2023.

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

// BookingsdSpec defines the desired state of Bookingsd
type BookingsdSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Size defines the number of Bookingsd instances
	// The following markers will use OpenAPI v3 schema to validate the value
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:validation:ExclusiveMaximum=false
	Size               int32  `json:"size"`
	ContainerPort      int32  `json:"containerPort"`
	DbServer           string `json:"dbServer"`
	DbPort             string `json:"dbPort"`
	DbUser             string `json:"dbUser"`
	DbName             string `json:"dbName"`
	MailhogHost        string `json:"mailhogHost,omitempty"`
	MailhogPort        string `json:"mailhogPort,omitempty"`
	InitContainerImage string `json:"initContainerImage"`
	ContainerImage     string `json:"containerImage"`
}

// BookingsdStatus defines the observed state of Bookingsd
type BookingsdStatus struct {
	// Represents the observations of a Bookingsd's current state.
	// Bookingsd.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// Bookingsd.status.conditions.status are one of True, False, Unknown.
	// Bookingsd.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// Bookingsd.status.conditions.Message is a human-readable message indicating details about the transition.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Bookingsd is the Schema for the bookingsds API
type Bookingsd struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BookingsdSpec   `json:"spec"`
	Status BookingsdStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// BookingsdList contains a list of Bookingsd
type BookingsdList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bookingsd `json:"items"`
}

type PostgresSpec struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1
	// +kubebuilder:validation:ExclusiveMaximum=false
	Size             int32  `json:"size"`
	ContainerPort    int32  `json:"containerPort"`
	StorageClassName string `json:"storageClassName"`
	StorageSize      string `json:"storageSize"`
	PostgresUser     string `json:"postgresUser"`
	PostgresPassword string `json:"postgresPassword"`
	PostgresDatabase string `json:"postgresDatabase"`
	ContainerImage   string `json:"containerImage"`
}

type PostgresStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Postgres struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresSpec   `json:"spec"`
	Status PostgresStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type PostgresList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Postgres `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Bookingsd{}, &BookingsdList{})
	SchemeBuilder.Register(&Postgres{}, &PostgresList{})
}
