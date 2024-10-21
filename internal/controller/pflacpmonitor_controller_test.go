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
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	pfstatusrelayv1alpha1 "github.com/openshift/pf-status-relay-operator/api/v1alpha1"
)

var _ = Describe("PFLACPMonitor Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName = "test-resource"
			timeout      = time.Second * 10
			interval     = time.Millisecond * 250

			dsImage = "quay.io/openshift/pf-status-relay:latest"
		)

		var dsName string
		var envVars []corev1.EnvVar

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		pflacpmonitor := &pfstatusrelayv1alpha1.PFLACPMonitor{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind PFLACPMonitor")
			dsName = fmt.Sprintf("pf-status-relay-daemonset-%s", typeNamespacedName.Name)
			envVars = []corev1.EnvVar{
				{
					Name:  "PF_STATUS_RELAY_INTERFACES",
					Value: "eth0",
				},
				{
					Name:  "PF_STATUS_RELAY_POLLING_INTERVAL",
					Value: "2000",
				},
			}

			err := k8sClient.Get(ctx, typeNamespacedName, pflacpmonitor)
			if err != nil && errors.IsNotFound(err) {
				resource := &pfstatusrelayv1alpha1.PFLACPMonitor{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: pfstatusrelayv1alpha1.PFLACPMonitorSpec{
						Interfaces: []string{
							"eth0",
						},
						PollingInterval: 2000,
						NodeSelector:    map[string]string{"key": "value"},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			err = os.Setenv("PF_STATUS_RELAY_IMAGE", dsImage)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			resource := &pfstatusrelayv1alpha1.PFLACPMonitor{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance PFLACPMonitor")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		Context("Deamonset validation", func() {
			var ds *appsv1.DaemonSet

			BeforeEach(func() {
				ds = &appsv1.DaemonSet{}
				Eventually(func() error {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: dsName, Namespace: typeNamespacedName.Namespace}, ds)
					return err
				}, timeout, interval).Should(Succeed())
			})

			It("creates a DeamonSet with correct config", func() {
				Expect(ds.Spec.Template.Spec.Containers[0].Image).To(Equal(dsImage))
				Expect(ds.Spec.Template.Spec.Containers[0].Env).To(Equal(envVars))
				Expect(ds.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"key": "value"}))
			})

			It("recreates the DeamonSet when this has been deleted", func() {
				By("deleting the DeamonSet")
				Expect(k8sClient.Delete(ctx, ds)).To(Succeed())

				Eventually(func() error {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: dsName, Namespace: typeNamespacedName.Namespace}, ds)
					return err
				}, timeout, interval).ShouldNot(Succeed())

				By("recreating the daemonset")
				Eventually(func() error {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: dsName, Namespace: typeNamespacedName.Namespace}, ds)
					return err
				}, timeout, interval).Should(Succeed())
			})

			It("restores the DeamonSet config when this has been externally modified", func() {
				By("modifying the ConfigMap")
				ds.Spec.Template.Spec.Containers[0].Env = nil
				Expect(k8sClient.Update(ctx, ds)).To(Succeed())

				By("checking the daemonset config")

				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: dsName, Namespace: typeNamespacedName.Namespace}, ds)
					Expect(err).NotTo(HaveOccurred())

					return reflect.DeepEqual(ds.Spec.Template.Spec.Containers[0].Env, envVars)
				}, timeout, interval).Should(BeTrue())
			})

			It("should modify Degraded status appropriately", func() {
				newName := "new-monitor"
				namespace := "default"

				By("creating a new PFLACPMonitor resource")
				newPFLACPMonitor := &pfstatusrelayv1alpha1.PFLACPMonitor{
					ObjectMeta: metav1.ObjectMeta{
						Name:      newName,
						Namespace: namespace,
					},
					Spec: pfstatusrelayv1alpha1.PFLACPMonitorSpec{
						Interfaces: []string{
							"eth1",
						},
					},
				}
				Expect(k8sClient.Create(ctx, newPFLACPMonitor)).To(Succeed())

				ds = &appsv1.DaemonSet{}
				Eventually(func() error {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("pf-status-relay-daemonset-%s", newName), Namespace: namespace}, ds)
					return err
				}, timeout, interval).Should(Succeed())

				By("Verifying degraded state is set to true if interfaces are the same")
				Eventually(func() error {
					monitor := &pfstatusrelayv1alpha1.PFLACPMonitor{}
					err := k8sClient.Get(ctx, types.NamespacedName{Name: newName, Namespace: namespace}, monitor)
					Expect(err).NotTo(HaveOccurred())

					monitor.Spec.Interfaces = []string{"eth0"}
					return k8sClient.Update(ctx, monitor)
				}, timeout, interval).Should(Succeed())

				Eventually(func() bool {
					monitor := &pfstatusrelayv1alpha1.PFLACPMonitor{}
					err := k8sClient.Get(ctx, types.NamespacedName{Name: newName, Namespace: namespace}, monitor)
					Expect(err).NotTo(HaveOccurred())

					return reflect.DeepEqual(monitor.Status.Degraded, true)
				}, timeout, interval).Should(BeTrue())

				ds = &appsv1.DaemonSet{}
				Eventually(func() error {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("pf-status-relay-daemonset-%s", newName), Namespace: namespace}, ds)
					return err
				}, timeout, interval).ShouldNot(Succeed())

				By("Updating the new PFLACPMonitor resource with different interfaces")
				Eventually(func() error {
					monitor := &pfstatusrelayv1alpha1.PFLACPMonitor{}
					err := k8sClient.Get(ctx, types.NamespacedName{Name: "new-monitor", Namespace: "default"}, monitor)
					Expect(err).NotTo(HaveOccurred())

					monitor.Spec.Interfaces = []string{"eth1"}
					return k8sClient.Update(ctx, monitor)
				}, timeout, interval).Should(Succeed())

				By("Verifying degraded state is set to false")
				Eventually(func() bool {
					monitor := &pfstatusrelayv1alpha1.PFLACPMonitor{}
					err := k8sClient.Get(ctx, types.NamespacedName{Name: "new-monitor", Namespace: "default"}, monitor)
					Expect(err).NotTo(HaveOccurred())

					return reflect.DeepEqual(monitor.Status.Degraded, false)
				}, timeout, interval).Should(BeTrue())

				ds = &appsv1.DaemonSet{}
				Eventually(func() error {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("pf-status-relay-daemonset-%s", "new-monitor"), Namespace: "default"}, ds)
					return err
				}, timeout, interval).Should(Succeed())
			})
		})
	})
})
