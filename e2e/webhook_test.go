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

package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pfv1alpha1 "github.com/openshift/pf-status-relay-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("webhook", Label("e2e", "webhook"), func() {

	It("rejects PFLACPMonitor with empty interface", func(ctx context.Context) {
		err := k8sClient.Create(ctx, &pfv1alpha1.PFLACPMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-empty-iface",
				Namespace: operatorNS,
			},
			Spec: pfv1alpha1.PFLACPMonitorSpec{
				Interfaces: []string{"eth0", " "},
			},
		})
		Expect(apierrors.IsInvalid(err)).To(BeTrue(), "expected Invalid, got: %v", err)
		Expect(err.Error()).To(ContainSubstring("interface cannot be empty"))
	})
})
