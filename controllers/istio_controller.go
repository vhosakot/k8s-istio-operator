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
	"context"

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

func (r *IstioReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	var Istio operatorv1alpha1.Istio

	r.Log.Info("inside Reconcile() function in istio_controller.go")

	if err := r.Get(ctx, req.NamespacedName, &Istio); err != nil {
		r.Log.Info("Istio CR deleted: ", "name", req.NamespacedName)
	} else {
		r.Log.Info("Istio CR created: ", "name", req.NamespacedName)
		r.Log.Info("Istio CR spec:", "spec", Istio.Spec)

		// read istio-init section from Istio CR
		r.Log.Info("istio-init", "chart", Istio.Spec.CcpIstioInit.Chart)
		r.Log.Info("istio-init", "values", Istio.Spec.CcpIstioInit.Values)

		// read istio section from Istio CR
		r.Log.Info("istio", "chart", Istio.Spec.CcpIstio.Chart)
		r.Log.Info("istio", "values", Istio.Spec.CcpIstio.Values)

		// read istio-remote from Istio CR
		r.Log.Info("istio-remote", "chart", Istio.Spec.CcpIstioRemote.Chart)
		r.Log.Info("istio-remote", "values", Istio.Spec.CcpIstioRemote.Values)
	}

	return ctrl.Result{}, nil
}

func (r *IstioReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Istio{}).
		Complete(r)
}
