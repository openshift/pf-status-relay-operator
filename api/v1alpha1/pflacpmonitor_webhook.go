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

package v1alpha1

import (
	"context"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const timeoutList = 60 * time.Second

// log is for logging in this package.
var pflacpmonitorlog = logf.Log.WithName("pflacpmonitor-resource")

type pflacpmonitorValidator struct {
	Client client.Client
}

var _ admission.CustomValidator = &pflacpmonitorValidator{}

func (r *PFLACPMonitor) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&pflacpmonitorValidator{Client: mgr.GetClient()}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-pfstatusrelay-openshift-io-v1alpha1-pflacpmonitor,mutating=false,failurePolicy=fail,sideEffects=None,groups=pfstatusrelay.openshift.io,resources=pflacpmonitors,verbs=create;update,versions=v1alpha1,name=vpflacpmonitor.kb.io,admissionReviewVersions=v1

// ValidateCreate implements admission.CustomValidator so a webhook will be registered for the type
func (v *pflacpmonitorValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*PFLACPMonitor)
	if !ok {
		return nil, apierrors.NewBadRequest("expected a PFLACPMonitor object")
	}

	pflacpmonitorlog.Info("validating create", "name", r.Name, "namespace", r.Namespace)
	return v.validate(ctx, r)
}

// ValidateUpdate implements admission.CustomValidator so a webhook will be registered for the type
func (v *pflacpmonitorValidator) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	r, ok := newObj.(*PFLACPMonitor)
	if !ok {
		return nil, apierrors.NewBadRequest("expected a PFLACPMonitor object")
	}

	pflacpmonitorlog.Info("validating update", "name", r.Name, "namespace", r.Namespace)
	return v.validate(ctx, r)
}

// ValidateDelete implements admission.CustomValidator so a webhook will be registered for the type
func (v *pflacpmonitorValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *pflacpmonitorValidator) validate(ctx context.Context, monitor *PFLACPMonitor) (admission.Warnings, error) {
	if err := monitor.validateSpec(); err != nil {
		return nil, err
	}

	if err := v.validateInterfaceUniqueness(ctx, monitor); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateInterfaceUniqueness checks if an interface is used by only one PFLACPMonitor
func (v *pflacpmonitorValidator) validateInterfaceUniqueness(ctx context.Context, monitor *PFLACPMonitor) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, timeoutList)
	defer cancel()

	monitorList := &PFLACPMonitorList{}
	err := v.Client.List(ctxTimeout, monitorList, client.InNamespace(monitor.Namespace))
	if err != nil {
		return apierrors.NewInternalError(err)
	}

	err = InterfaceUniqueness(monitor, monitorList)
	if err != nil {
		return apierrors.NewConflict(schema.GroupResource{Group: GroupVersion.Group, Resource: "pflacpmonitors"}, monitor.Name, err)
	}

	return nil
}

func (r *PFLACPMonitor) validateSpec() error {
	var allErrs field.ErrorList
	if err := validateInterfaces(r.Spec.Interfaces); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(GroupVersion.WithKind("PFLACPMonitor").GroupKind(), r.Name, allErrs)
	}

	return nil
}

func validateInterfaces(interfaces []string) *field.Error {
	pfs := make([]string, 0, len(interfaces))

	for i := range interfaces {
		pf := strings.TrimSpace(interfaces[i])
		if pf == "" {
			return field.Invalid(field.NewPath("spec").Child("interfaces").Index(i), interfaces[i], "interface cannot be empty")
		}

		pfs = append(pfs, pf)
	}

	checkUnique := make(map[string]struct{})
	for _, pf := range pfs {
		if _, ok := checkUnique[pf]; ok {
			return field.Invalid(field.NewPath("spec").Child("interfaces"), pfs, "interfaces must be unique")
		}
		checkUnique[pf] = struct{}{}
	}

	return nil
}
