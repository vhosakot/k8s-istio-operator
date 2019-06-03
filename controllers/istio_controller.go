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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/go-logr/logr"
	apiextclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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
	charts_dir := os.Getenv("CHARTS_PATH")
	r.Log.Info(fmt.Sprintf("CHARTS_PATH: %s", charts_dir))
	if len(charts_dir) == 0 {
		r.Log.Info("environment variable CHARTS_PATH not set")
	}

	if err := r.Get(ctx, req.NamespacedName, &Istio); err != nil {
		r.Log.Info(fmt.Sprintf("Istio CR deleted: %s", req.NamespacedName.String()))
	} else {
		r.Log.Info(fmt.Sprintf("Istio CR created: %s", req.NamespacedName.String()))
		r.Log.Info("Istio CR spec:", "spec", Istio.Spec)

		// validate istio CR spec
		if !r.IstioCRSpecIsValid(Istio) {
			return ctrl.Result{}, nil
		}

		// generate values file needed for helm
		r.GenerateValuesYamlFromIstioSpec("istio-init", Istio.Spec.CcpIstioInit.Values)
		r.GenerateValuesYamlFromIstioSpec("istio", Istio.Spec.CcpIstio.Values)
		r.GenerateValuesYamlFromIstioSpec("istio-remote", Istio.Spec.CcpIstioRemote.Values)

		// delete istio if it already exists
		r.Log.Info("deleting istio if it already exists.")
		if err := r.DeleteIstio(); err != nil {
			r.UpdateIstioCR(ctx, &Istio, err.Error())
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// update istio CR spec
func (r *IstioReconciler) UpdateIstioCR(ctx context.Context, ist *operatorv1alpha1.Istio, status string) {
	ist.Status.Active = status
	r.Log.Info(fmt.Sprintf("Istio CR status: %s", ist.Status.Active))
	if err := r.Update(ctx, ist); err != nil {
		r.Log.Error(err,
			fmt.Sprintf("unable to update Istio CR's status to \"%s\".", status))
	}
}

// delete istio, istio-init, istio's CRDs and jobs
func (r *IstioReconciler) DeleteIstio() error {
	cmd := fmt.Sprintf("helm ls | grep \"%s \"", operatorv1alpha1.IstioHelmChartName)
	if out, _ := r.RunCommand(cmd); len(out) == 0 {
		r.Log.Info(fmt.Sprintf("%s helm chart not found.", operatorv1alpha1.IstioHelmChartName))
	} else {
		// delete istio helm chart
		cmd := fmt.Sprintf("helm delete --purge %s", operatorv1alpha1.IstioHelmChartName)
		if _, err := r.RunCommand(cmd); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				e := fmt.Sprintf("Failed to delete %s helm chart, error: %s, %s",
					operatorv1alpha1.IstioHelmChartName, string(exitErr.Stderr), err)
				return errors.New(e)
			}
		} else {
			r.Log.Info(fmt.Sprintf("%s helm chart deleted", operatorv1alpha1.IstioHelmChartName))
		}
	}

	cmd = fmt.Sprintf("helm ls | grep \"%s \"", operatorv1alpha1.IstioInitHelmChartName)
	if out, _ := r.RunCommand(cmd); len(out) == 0 {
		r.Log.Info(fmt.Sprintf("%s helm chart not found.", operatorv1alpha1.IstioInitHelmChartName))
	} else {
		// delete istio-init helm chart
		cmd := fmt.Sprintf("helm delete --purge %s", operatorv1alpha1.IstioInitHelmChartName)
		if _, err := r.RunCommand(cmd); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				e := fmt.Sprintf("Failed to delete %s helm chart, error: %s, %s",
					operatorv1alpha1.IstioInitHelmChartName, string(exitErr.Stderr), err)
				return errors.New(e)
			}
		} else {
			r.Log.Info(fmt.Sprintf("%s helm chart deleted", operatorv1alpha1.IstioInitHelmChartName))
		}
	}

	// delete all istio CRDs
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	clientset, err := apiextclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	crdList, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().List(v1.ListOptions{LabelSelector: ""})
	if err != nil {
		return err
	}
	for _, crd := range crdList.Items {
		if strings.Contains(crd.Spec.Group, operatorv1alpha1.IstioCRDGroupSuffix) {
			istioCRName := fmt.Sprintf("%s.%s", crd.Spec.Names.Plural, crd.Spec.Group)
			if err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(istioCRName, nil); err != nil {
				return err
			} else {
				r.Log.Info(fmt.Sprintf("istio CR %s deleted", istioCRName))
			}
		}
	}

	return nil
}

// run linux shell command, return output and error, assumes bash shell
func (r *IstioReconciler) RunCommand(cmd string) ([]byte, error) {
	r.Log.Info(fmt.Sprintf("running command: %s", cmd))
	out, err := exec.Command("bash", "-c", cmd).Output()
	r.Log.Info(fmt.Sprintf("output: %s", string(out)))
	return out, err
}

// validate Istio CR spec
func (r *IstioReconciler) IstioCRSpecIsValid(ist operatorv1alpha1.Istio) bool {
	// read istio-init section from Istio CR
	r.Log.Info("istio-init", "chart", ist.Spec.CcpIstioInit.Chart)
	r.Log.Info("istio-init", "values", ist.Spec.CcpIstioInit.Values)
	if ist.Spec.CcpIstioInit.Chart == "" {
		r.Log.Error(errors.New("invalid istio CR spec"),
			"istio-init helm chart is empty in istio CR spec, cannot install istio-init and istio.")
		return false
	}
	if _, err := os.Stat(ist.Spec.CcpIstioInit.Chart); os.IsNotExist(err) &&
		!strings.HasPrefix(ist.Spec.CcpIstioInit.Chart, "http") {
		e := fmt.Sprintf("istio-init helm chart %s %s", ist.Spec.CcpIstioInit.Chart,
			"does not exist.")
		r.Log.Error(errors.New(e), e)
		return false
	}

	// read istio section from Istio CR
	r.Log.Info("istio", "chart", ist.Spec.CcpIstio.Chart)
	r.Log.Info("istio", "values", ist.Spec.CcpIstio.Values)
	if ist.Spec.CcpIstio.Chart == "" {
		r.Log.Error(errors.New("invalid istio CR spec"),
			"istio helm chart is empty in istio CR spec, cannot install istio.")
		return false
	}
	if _, err := os.Stat(ist.Spec.CcpIstio.Chart); os.IsNotExist(err) &&
		!strings.HasPrefix(ist.Spec.CcpIstio.Chart, "http") {
		e := fmt.Sprintf("istio helm chart %s %s", ist.Spec.CcpIstio.Chart,
			"does not exist.")
		r.Log.Error(errors.New(e), e)
		return false
	}

	// read istio-remote section from Istio CR
	r.Log.Info("istio-remote", "chart", ist.Spec.CcpIstioRemote.Chart)
	r.Log.Info("istio-remote", "values", ist.Spec.CcpIstioRemote.Values)

	return true
}

// generate values file needed to install istio using helm
func (r *IstioReconciler) GenerateValuesYamlFromIstioSpec(chartName string, values string) {
	if values == "" {
		r.Log.Info(fmt.Sprintf("values not found for %s %s", chartName, "in Istio CR spec."))
	} else {
		f := []byte(values)
		f = append(f, "\n"...)
		valuesFileName := fmt.Sprintf("%s%s", chartName, "-values.yaml")
		os.Remove(valuesFileName)
		err := ioutil.WriteFile(valuesFileName, f, 0644)
		if err != nil {
			r.Log.Error(err, fmt.Sprintf("Failed to generate values file for %s %s",
				chartName, "in Istio CR spec."))
			return
		}
		r.Log.Info(fmt.Sprintf("Generated values file %s %s %s %s", valuesFileName, "for",
			chartName, "in Istio CR spec."))
	}
}

func (r *IstioReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Istio{}).
		Complete(r)
}
