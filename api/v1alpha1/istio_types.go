/*

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

const (
	IstioHelmChartName     = "istio"
	IstioInitHelmChartName = "istio-init"
)

// IstioInitValues defines the istio-init section in Istio CR spec
type IstioInitValues struct {
	Chart  string `json:"chart,omitempty"`
	Values string `json:"values,omitempty"`
}

// IstioValues defines the istio section in Istio CR spec
type IstioValues struct {
	Chart  string `json:"chart,omitempty"`
	Values string `json:"values,omitempty"`
}

// IstioRemoteValues defines the istio-remote section in Istio CR spec
type IstioRemoteValues struct {
	Chart  string `json:"chart,omitempty"`
	Values string `json:"values,omitempty"`
}

// IstioSpec defines the desired state of Istio
type IstioSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	CcpIstioInit   IstioInitValues   `json:"istio-init,omitempty"`
	CcpIstio       IstioValues       `json:"istio,omitempty"`
	CcpIstioRemote IstioRemoteValues `json:"istio-remote,omitempty"`
}

// IstioStatus defines the observed state of Istio
type IstioStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Active string `json:"active,omitempty"`
}

// +kubebuilder:object:root=true

// Istio is the Schema for the istios API
type Istio struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IstioSpec   `json:"spec,omitempty"`
	Status IstioStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IstioList contains a list of Istio
type IstioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Istio `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Istio{}, &IstioList{})
}
