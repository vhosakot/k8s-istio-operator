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

package controllers

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "wwwin-github.cisco.com/CPSG/ccp-istio-operator/api/v1alpha1"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

// IstioReconciler reconciles a Istio object
type IstioReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=operator.ccp.cisco.com,resources=istios,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.ccp.cisco.com,resources=istios/status,verbs=get;update;patch

// Function to join all strings in a slice of strings, helper function for logging
func JoinStrings(strSlice []string) string {
	var b bytes.Buffer
	for _, str := range strSlice {
		b.WriteString(str)
	}
	return b.String()
}

func (r *IstioReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	var Istio operatorv1alpha1.Istio

	r.Log.Info("inside Reconcile() function in istio_controller.go")
	charts_dir := os.Getenv("CHARTS_PATH")
	r.Log.Info(JoinStrings([]string{"CHARTS_PATH: ", charts_dir}))
	if len(charts_dir) == 0 {
		r.Log.Info("environment variable CHARTS_PATH not set")
	}

	if err := r.Get(ctx, req.NamespacedName, &Istio); err != nil {
		r.Log.Info(JoinStrings([]string{"Istio CR deleted: ", req.NamespacedName.String()}))
	} else {
		r.Log.Info(JoinStrings([]string{"Istio CR created: ", req.NamespacedName.String()}))
		r.Log.Info("Istio CR spec:", "spec", Istio.Spec)

		// read istio-init section from Istio CR
		r.Log.Info("istio-init", "chart", Istio.Spec.CcpIstioInit.Chart)
		r.Log.Info("istio-init", "values", Istio.Spec.CcpIstioInit.Values)
		r.GenerateValuesYamlFromIstioSpec("istio-init", Istio.Spec.CcpIstioInit.Values)

		// read istio section from Istio CR
		r.Log.Info("istio", "chart", Istio.Spec.CcpIstio.Chart)
		r.Log.Info("istio", "values", Istio.Spec.CcpIstio.Values)
		r.GenerateValuesYamlFromIstioSpec("istio", Istio.Spec.CcpIstio.Values)

		// read istio-remote section from Istio CR
		r.Log.Info("istio-remote", "chart", Istio.Spec.CcpIstioRemote.Chart)
		r.Log.Info("istio-remote", "values", Istio.Spec.CcpIstioRemote.Values)
		r.GenerateValuesYamlFromIstioSpec("istio-remote", Istio.Spec.CcpIstioRemote.Values)
	}

	return ctrl.Result{}, nil
}

// Function to generate values file needed to install istio using helm
func (r *IstioReconciler) GenerateValuesYamlFromIstioSpec(chartName string, values string) {
	if values == "" {
		r.Log.Info(JoinStrings([]string{"values not found for ", chartName, " in Istio CR spec."}))
	} else {
		f := []byte(values)
		f = append(f, "\n"...)
		valuesFileName := JoinStrings([]string{chartName, "-values.yaml"})
		os.Remove(valuesFileName)
		err := ioutil.WriteFile(valuesFileName, f, 0644)
		if err != nil {
			r.Log.Error(err, JoinStrings([]string{"Failed to generate values file for ",
				chartName, " in Istio CR spec."}))
			return
		}
		r.Log.Info(JoinStrings([]string{"Generated values file ", valuesFileName, " for ",
			chartName, " in Istio CR spec."}))
	}
}

func (r *IstioReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Istio{}).
		Complete(r)
}
