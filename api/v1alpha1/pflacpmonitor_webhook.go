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
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var pflacpmonitorlog = logf.Log.WithName("pflacpmonitor-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *PFLACPMonitor) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-pfstatusrelay-openshift-io-v1alpha1-pflacpmonitor,mutating=false,failurePolicy=fail,sideEffects=None,groups=pfstatusrelay.openshift.io,resources=pflacpmonitors,verbs=create;update,versions=v1alpha1,name=vpflacpmonitor.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &PFLACPMonitor{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PFLACPMonitor) ValidateCreate() (admission.Warnings, error) {
	pflacpmonitorlog.Info("validation create", "name", r.Name)

	return nil, r.validatePFLACPMonitor()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PFLACPMonitor) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	pflacpmonitorlog.Info("validation update", "name", r.Name)

	return nil, r.validatePFLACPMonitor()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PFLACPMonitor) ValidateDelete() (admission.Warnings, error) {
	pflacpmonitorlog.Info("validation delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

func (r *PFLACPMonitor) validatePFLACPMonitor() error {
	var allErrs field.ErrorList
	if err := validateInterfaces(r.Spec.Interfaces); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(GroupVersion.WithKind("PFLACPMonitor").GroupKind(), r.Name, allErrs)
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
