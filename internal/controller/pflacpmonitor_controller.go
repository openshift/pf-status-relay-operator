/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	"os"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	pfstatusrelayv1alpha1 "github.com/mlguerrero12/pf-status-relay-operator/api/v1alpha1"
	"github.com/mlguerrero12/pf-status-relay-operator/internal/log"
	"github.com/mlguerrero12/pf-status-relay-operator/internal/validation"
)

// PFLACPMonitorReconciler reconciles a PFLACPMonitor object
type PFLACPMonitorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pfstatusrelay.openshift.io,resources=pflacpmonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pfstatusrelay.openshift.io,resources=pflacpmonitors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pfstatusrelay.openshift.io,resources=pflacpmonitors/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PFLACPMonitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Log.Info("reconciling PFLACPMonitor", "name", req.Name, "namespace", req.Namespace)

	pfMonitor := &pfstatusrelayv1alpha1.PFLACPMonitor{}
	err := r.Get(ctx, req.NamespacedName, pfMonitor)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Log.Debug("PFLACPMonitor not found, ignoring")
			return ctrl.Result{}, nil
		}
		log.Log.Error("unable to get PFLACPMonitor", "error", err)
		return ctrl.Result{}, err
	}

	pfMonitorList := &pfstatusrelayv1alpha1.PFLACPMonitorList{}
	err = r.List(ctx, pfMonitorList)
	if err != nil {
		log.Log.Error("unable to list PFLACPMonitor", "error", err)
		return ctrl.Result{}, err
	}

	err = validation.InterfaceUniqueness(pfMonitor, pfMonitorList)
	if err != nil {
		if pfMonitor.Status.Degraded {
			return ctrl.Result{}, nil
		}

		log.Log.Error("failed to validate PFLACPMonitor", "error", err)

		pfMonitor.Status.Degraded = true
		pfMonitor.Status.ErrorMessage = err.Error()

		if err = r.Status().Update(ctx, pfMonitor); err != nil {
			log.Log.Error("failed to update status", "error", err)
			return ctrl.Result{}, err
		}

		// Delete daemonset if exists
		err = r.deleteDaemonSet(ctx, pfMonitor)
		if err != nil {
			log.Log.Error("failed to delete daemonset", "error", err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	err = r.syncDaemonSet(ctx, pfMonitor)
	if err != nil {
		log.Log.Error("failed to sync daemonset", "error", err)
		return ctrl.Result{}, err
	}

	if pfMonitor.Status.Degraded {
		pfMonitor.Status.Degraded = false
		pfMonitor.Status.ErrorMessage = ""

		if err = r.Status().Update(ctx, pfMonitor); err != nil {
			log.Log.Error("failed to update status", "error", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *PFLACPMonitorReconciler) syncDaemonSet(ctx context.Context, pfMonitor *pfstatusrelayv1alpha1.PFLACPMonitor) error {
	log.Log.Info("syncing daemonset", "name", pfMonitor.Name, "namespace", pfMonitor.Namespace)

	name := fmt.Sprintf("pf-status-relay-daemonset-%s", pfMonitor.Name)
	image, err := getDaemonSetImage()
	if err != nil {
		return err
	}

	refDs := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pfMonitor.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					HostNetwork:  true,
					HostPID:      true,
					NodeSelector: pfMonitor.Spec.NodeSelector,
					Containers: []corev1.Container{
						{
							Name:  "pf-status-relay",
							Image: image,
							SecurityContext: &corev1.SecurityContext{
								Privileged: func(b bool) *bool { return &b }(true),
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PF_STATUS_RELAY_INTERFACES",
									Value: strings.Join(pfMonitor.Spec.Interfaces, ","),
								},
								{
									Name:  "PF_STATUS_RELAY_POLLING_INTERVAL",
									Value: fmt.Sprintf("%d", pfMonitor.Spec.PollingInterval),
								},
							},
						},
					},
				},
			},
		},
	}

	ds := &appsv1.DaemonSet{}
	err = r.Get(ctx, types.NamespacedName{Name: name, Namespace: pfMonitor.Namespace}, ds)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Log.Info("daemon set not found, creating", "name", name)

			if err = controllerutil.SetControllerReference(pfMonitor, refDs, r.Scheme); err != nil {
				return fmt.Errorf("failed to set controller reference: %w", err)
			}

			if err = r.Create(ctx, refDs); err != nil {
				return fmt.Errorf("failed to create daemon set: %w", err)
			}

			return nil
		}

		return fmt.Errorf("failed to get daemon set: %w", err)
	}

	if !equality.Semantic.DeepEqual(ds.Spec, refDs.Spec) {
		log.Log.Info("daemon set found, updating", "name", name)

		ds.Spec = refDs.Spec
		if err = r.Update(ctx, ds); err != nil {
			return fmt.Errorf("failed to update daemon set: %w", err)
		}

		log.Log.Debug("daemon set updated", "name", name)
	}

	log.Log.Debug("daemon set already up to date", "name", name)
	return nil
}

func (r *PFLACPMonitorReconciler) deleteDaemonSet(ctx context.Context, pfMonitor *pfstatusrelayv1alpha1.PFLACPMonitor) error {
	name := fmt.Sprintf("pf-status-relay-daemonset-%s", pfMonitor.Name)
	ds := &appsv1.DaemonSet{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: pfMonitor.Namespace}, ds)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return err
	}

	return r.Delete(ctx, ds)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PFLACPMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pfstatusrelayv1alpha1.PFLACPMonitor{}).
		Owns(&appsv1.DaemonSet{}).
		Complete(r)
}

func getDaemonSetImage() (string, error) {
	var pfStatusRelayImageEnvVar = "PF_STATUS_RELAY_IMAGE"

	ns, found := os.LookupEnv(pfStatusRelayImageEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", pfStatusRelayImageEnvVar)
	}
	return ns, nil
}
