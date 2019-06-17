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
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apiextclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
		// delete istio
		r.Log.Info("deleting istio")
		if err := r.DeleteIstio(); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if Istio.Status.ObservedGeneration != Istio.ObjectMeta.Generation {
			// this if branch is hit when metadata.generation in istio CR is incremented
			if Istio.Status.ObservedGeneration == 0 && Istio.ObjectMeta.Generation == 1 {
				r.Log.Info(fmt.Sprintf("New Istio CR created: %s", req.NamespacedName.String()))
			} else {
				// CR is updated using:
				// "kubectl edit istio <name of istio CR>" or
				// "kubectl apply -f <updated CR manifest file>
				r.Log.Info(fmt.Sprintf("Istio CR updated: %s", req.NamespacedName.String()))
			}
			r.Log.Info(fmt.Sprintf("  metadata.generation = %s",
				strconv.FormatInt(Istio.ObjectMeta.Generation, 10)))
			r.Log.Info(fmt.Sprintf("  status.observedGeneration = %s",
				strconv.FormatInt(Istio.Status.ObservedGeneration, 10)))

			// update ObservedGeneration and Version in CR status
			Istio.Status.ObservedGeneration = Istio.ObjectMeta.Generation
			istioVersion := strings.Split(Istio.Spec.CcpIstio.Chart, "/")
			Istio.Status.Version = istioVersion[len(istioVersion)-1]
			r.Status().Update(ctx, &Istio)

			r.Log.Info("Istio CR spec: ", "spec", Istio.Spec)

			// validate istio CR spec
			if !r.IstioCRSpecIsValid(Istio) {
				r.UpdateIstioCRStatus(ctx, &Istio, "InvalidIstioCRSpec")
				return ctrl.Result{}, nil
			}

			// generate values file needed for helm
			r.UpdateIstioCRStatus(ctx, &Istio, "GeneratingHelmValuesFile")
			r.GenerateValuesYamlFromIstioSpec("istio-init", Istio.Spec.CcpIstioInit.Values)
			r.GenerateValuesYamlFromIstioSpec("istio", Istio.Spec.CcpIstio.Values)
			r.GenerateValuesYamlFromIstioSpec("istio-remote", Istio.Spec.CcpIstioRemote.Values)

			// delete istio if it already exists
			r.UpdateIstioCRStatus(ctx, &Istio, "CleaningIstioPreinstall")
			r.Log.Info("deleting istio if it already exists.")
			if err := r.DeleteIstio(); err != nil {
				r.UpdateIstioCRStatus(ctx, &Istio, "PreinstallCleanupFailed")
				return ctrl.Result{}, err
			}

			// TODO: Instead of sleeping below, add post-delete steps here to check if
			// all the istio pods, CRDs and jobs are deleted before installing istio
			time.Sleep(10 * time.Second)

			// install istio
			r.Log.Info("installing istio")
			r.UpdateIstioCRStatus(ctx, &Istio, "InstallingIstio")
			if err := r.InstallIstio(Istio.Spec); err != nil {
				r.UpdateIstioCRStatus(ctx, &Istio, "InstallationFailed")
				return ctrl.Result{}, err
			}

			r.UpdateIstioCRStatus(ctx, &Istio, "PostInstallChecks")
			if err := r.DoPostInstallChecks(); err != nil {
				r.UpdateIstioCRStatus(ctx, &Istio, "PostInstallChecksFailed")
				r.Log.Error(err, "PostInstallChecksFailed")
			} else {
				r.UpdateIstioCRStatus(ctx, &Istio, "IstioInstalledActive")
			}
		} else {
			// this else branch is hit when metadata.generation in istio CR is not incremented (when
			// istio CR's status is updated)
			//
			// no need to reconcile istio when istio CR's status is updated,
			// istio needs to be reconciled only when the CR's spec is updated and
			// NOT when the CR's status is updated (when r.Status().Update(ctx, ist) is done in UpdateIstioCRStatus())
			//
			// https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/ says:
			//
			//  The .metadata.generation value is incremented for all changes, except for changes to .metadata or .status.
			r.Log.Info("Istio CR status: ", "status", Istio.Status)
		}
	}
	return ctrl.Result{}, nil
}

// check post install if all istio pods have reached Running and Ready state or Completed state
func (r *IstioReconciler) DoPostInstallChecks() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.New(fmt.Sprintf("%s, %s", "post-install check failed", err.Error()))
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.New(fmt.Sprintf("%s, %s", "post-install check failed", err.Error()))
	}

	for i := 1; i <= operatorv1alpha1.TimeoutInternal/5; i++ {
		time.Sleep(5 * time.Second)
		allIstioPodsAreGood := true
		podList, err := clientset.CoreV1().Pods(operatorv1alpha1.IstioNamespace).List(v1.ListOptions{})
		if err != nil {
			return errors.New(fmt.Sprintf("%s, %s", "post-install check failed", err.Error()))
		}

		for _, pod := range podList.Items {
			for _, containerStatus := range pod.Status.ContainerStatuses {
				// check if pod has reached Running and Ready state
				if containerStatus.State.Running != nil && containerStatus.Ready {
					continue
					// check if job's pod completed successfully
				} else if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.Reason == "Completed" {
					continue
				} else {
					r.Log.Info(fmt.Sprintf("%s container in %s pod did not "+
						"reach Ready or Completed state, "+
						"will check again after 5 seconds...",
						containerStatus.Name, pod.ObjectMeta.Name))
					allIstioPodsAreGood = false
					continue
				}
			}
			if pod.Status.Phase != "Running" && pod.Status.Phase != "Succeeded" {
				r.Log.Info(fmt.Sprintf("%s pod did not reach Running or Succeeded phase, "+
					"will check again after 5 seconds...", pod.ObjectMeta.Name))
				allIstioPodsAreGood = false
				continue
			}
		}
		if i == operatorv1alpha1.TimeoutInternal/5 {
			return errors.New(fmt.Sprintf("post-install checks timed out after %s seconds and failed, "+
				"istio pod(s) did not reach Running and Ready state or Completed state",
				strconv.FormatInt(operatorv1alpha1.TimeoutInternal, 10)))
		} else if allIstioPodsAreGood {
			return nil
		}
	}
	return nil
}

// update istio CR's status.active field
func (r *IstioReconciler) UpdateIstioCRStatus(ctx context.Context, ist *operatorv1alpha1.Istio, status string) {
	ist.Status.Active = status

	// updating istio CR's status below (r.Status().Update(ctx, ist)) does not
	// increment metadata.generation in istio CR
	if err := r.Status().Update(ctx, ist); err != nil {
		r.Log.Error(err, fmt.Sprintf("unable to update Istio CR status to \"%s\".", status))
		return
	} else {
		r.Log.Info(fmt.Sprintf("Istio CR status updated to: %s", ist.Status.Active))
	}
}

// install istio-init and istio
func (r *IstioReconciler) InstallIstio(istSpec operatorv1alpha1.IstioSpec) error {
	cmd := ""
	// install istio-init
	if istSpec.CcpIstioInit.Values == "" {
		cmd = fmt.Sprintf("helm install %s %s %s %s %s", istSpec.CcpIstioInit.Chart,
			" --name ", operatorv1alpha1.IstioInitHelmChartName, " --namespace ",
			operatorv1alpha1.IstioNamespace)
	} else {
		cmd = fmt.Sprintf("helm install %s %s %s %s %s %s", istSpec.CcpIstioInit.Chart, " -f istio-init-values.yaml",
			" --name ", operatorv1alpha1.IstioInitHelmChartName, " --namespace ",
			operatorv1alpha1.IstioNamespace)
	}
	if _, err := r.RunCommand(cmd); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			e := fmt.Sprintf("Failed to install %s helm chart, error: %s, %s",
				operatorv1alpha1.IstioInitHelmChartName, string(exitErr.Stderr), err)
			return errors.New(e)
		}
	} else {
		r.Log.Info(fmt.Sprintf("%s helm chart installed", operatorv1alpha1.IstioInitHelmChartName))
	}

	if err := r.DoPostInstallChecks(); err != nil {
		return err
	}

	// install istio
	if istSpec.CcpIstioInit.Values == "" {
		cmd = fmt.Sprintf("helm install %s %s %s %s %s", istSpec.CcpIstio.Chart,
			" --name ", operatorv1alpha1.IstioHelmChartName, " --namespace ",
			operatorv1alpha1.IstioNamespace)
	} else {
		cmd = fmt.Sprintf("helm install %s %s %s %s %s %s", istSpec.CcpIstio.Chart, " -f istio-values.yaml",
			" --name ", operatorv1alpha1.IstioHelmChartName, " --namespace ",
			operatorv1alpha1.IstioNamespace)
	}
	if _, err := r.RunCommand(cmd); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			e := fmt.Sprintf("Failed to install %s helm chart, error: %s, %s",
				operatorv1alpha1.IstioHelmChartName, string(exitErr.Stderr), err)
			return errors.New(e)
		}
	} else {
		r.Log.Info(fmt.Sprintf("%s helm chart installed", operatorv1alpha1.IstioHelmChartName))
	}
	return nil
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
		return errors.New(fmt.Sprintf("%s, %s", "failed to delete istio CRDs", err.Error()))
	}
	extclientset, err := apiextclientset.NewForConfig(config)
	if err != nil {
		return errors.New(fmt.Sprintf("%s, %s", "failed to delete istio CRDs", err.Error()))
	}
	crdList, err := extclientset.ApiextensionsV1beta1().CustomResourceDefinitions().List(v1.ListOptions{})
	if err != nil {
		return errors.New(fmt.Sprintf("%s, %s", "failed to delete istio CRDs", err.Error()))
	}
	for _, crd := range crdList.Items {
		if strings.Contains(crd.ObjectMeta.Name, operatorv1alpha1.IstioCRDGroupSuffix) {
			if err = extclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.ObjectMeta.Name, nil); err != nil {
				return errors.New(fmt.Sprintf("%s, %s", "failed to delete istio CRDs", err.Error()))
			} else {
				r.Log.Info(fmt.Sprintf("istio CRD %s deleted", crd.ObjectMeta.Name))
			}
		}
	}

	// delete all istio jobs
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.New(fmt.Sprintf("%s, %s", "failed to delete istio jobs", err.Error()))
	}
	jobList, err := clientset.BatchV1().Jobs(operatorv1alpha1.IstioNamespace).List(v1.ListOptions{})
	if err != nil {
		return errors.New(fmt.Sprintf("%s, %s", "failed to delete istio jobs", err.Error()))
	}
	for _, job := range jobList.Items {
		if err = clientset.BatchV1().Jobs(operatorv1alpha1.IstioNamespace).Delete(job.ObjectMeta.Name, nil); err != nil {
			return errors.New(fmt.Sprintf("%s, %s", "failed to delete istio jobs", err.Error()))
		}
		r.Log.Info(fmt.Sprintf("istio job %s deleted", job.ObjectMeta.Name))
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
