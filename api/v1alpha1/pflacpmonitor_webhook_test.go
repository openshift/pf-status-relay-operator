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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("PFLACPMonitor Webhook", func() {
	var validator *pflacpmonitorValidator
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		// We need to register our custom resource type with the fake client's scheme
		scheme := runtime.NewScheme()
		err := AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())

		// Create a fake client for our tests
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		validator = &pflacpmonitorValidator{Client: fakeClient}
	})

	Describe("ValidateCreate", func() {
		Context("with invalid spec", func() {
			It("should reject an empty interface", func() {
				monitor := &PFLACPMonitor{
					ObjectMeta: metav1.ObjectMeta{Name: "test-monitor", Namespace: "default"},
					Spec: PFLACPMonitorSpec{
						Interfaces: []string{"eth0", " "}, // Contains an empty string after trim
					},
				}
				_, err := validator.ValidateCreate(ctx, monitor)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("interface cannot be empty"))
			})

			It("should reject duplicate interfaces within the same resource", func() {
				monitor := &PFLACPMonitor{
					ObjectMeta: metav1.ObjectMeta{Name: "test-monitor", Namespace: "default"},
					Spec: PFLACPMonitorSpec{
						Interfaces: []string{"eth0", "eth1", "eth0"},
					},
				}
				_, err := validator.ValidateCreate(ctx, monitor)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("interfaces must be unique"))
			})
		})

		Context("with conflicting existing resources", func() {
			BeforeEach(func() {
				// Pre-populate the fake client with an existing monitor
				existingMonitor := &PFLACPMonitor{
					ObjectMeta: metav1.ObjectMeta{Name: "existing-monitor", Namespace: "default"},
					Spec: PFLACPMonitorSpec{
						Interfaces: []string{"eth0", "eth1"},
					},
				}
				err := validator.Client.Create(ctx, existingMonitor)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject a new monitor that uses an already claimed interface", func() {
				newMonitor := &PFLACPMonitor{
					ObjectMeta: metav1.ObjectMeta{Name: "new-monitor", Namespace: "default"},
					Spec: PFLACPMonitorSpec{
						Interfaces: []string{"eth2", "eth0"}, // eth0 is already used
					},
				}
				_, err := validator.ValidateCreate(ctx, newMonitor)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("interfaces [eth2 eth0] conflict with the ones from PFLACPMonitor existing-monitor"))
			})

			It("should allow a new monitor with unique interfaces", func() {
				newMonitor := &PFLACPMonitor{
					ObjectMeta: metav1.ObjectMeta{Name: "new-monitor", Namespace: "default"},
					Spec: PFLACPMonitorSpec{
						Interfaces: []string{"eth2", "eth3"},
					},
				}
				_, err := validator.ValidateCreate(ctx, newMonitor)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with a valid spec and no conflicts", func() {
			It("should successfully validate the resource", func() {
				monitor := &PFLACPMonitor{
					ObjectMeta: metav1.ObjectMeta{Name: "test-monitor", Namespace: "default"},
					Spec: PFLACPMonitorSpec{
						Interfaces: []string{"eth0", "eth1"},
					},
				}
				_, err := validator.ValidateCreate(ctx, monitor)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("ValidateUpdate", func() {
		var oldMonitor *PFLACPMonitor

		BeforeEach(func() {
			// Create a monitor that will be "updated"
			oldMonitor = &PFLACPMonitor{
				ObjectMeta: metav1.ObjectMeta{Name: "monitor-to-update", Namespace: "default"},
				Spec: PFLACPMonitorSpec{
					Interfaces: []string{"eth0"},
				},
			}
			err := validator.Client.Create(ctx, oldMonitor)
			Expect(err).NotTo(HaveOccurred())

			// Create another monitor that causes a potential conflict
			conflictingMonitor := &PFLACPMonitor{
				ObjectMeta: metav1.ObjectMeta{Name: "conflicting-monitor", Namespace: "default"},
				Spec: PFLACPMonitorSpec{
					Interfaces: []string{"eth1"},
				},
			}
			err = validator.Client.Create(ctx, conflictingMonitor)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject an update that introduces a conflict with another monitor", func() {
			updatedMonitor := oldMonitor.DeepCopy()
			updatedMonitor.Spec.Interfaces = []string{"eth0", "eth1"} // eth1 is used by conflicting-monitor

			_, err := validator.ValidateUpdate(ctx, oldMonitor, updatedMonitor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("interfaces [eth0 eth1] conflict with the ones from PFLACPMonitor conflicting-monitor"))
		})

		It("should allow an update that does not introduce any conflicts", func() {
			updatedMonitor := oldMonitor.DeepCopy()
			updatedMonitor.Spec.Interfaces = []string{"eth0", "eth2"} // eth2 is available

			_, err := validator.ValidateUpdate(ctx, oldMonitor, updatedMonitor)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should allow an update that does not change the interfaces", func() {
			updatedMonitor := oldMonitor.DeepCopy() // No changes to spec

			_, err := validator.ValidateUpdate(ctx, oldMonitor, updatedMonitor)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject an update that introduces duplicate interfaces", func() {
			updatedMonitor := oldMonitor.DeepCopy()
			updatedMonitor.Spec.Interfaces = []string{"eth0", "eth0"}

			_, err := validator.ValidateUpdate(ctx, oldMonitor, updatedMonitor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("interfaces must be unique"))
		})
	})
})
