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
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const metricsURL = `https://pf-status-relay-operator-controller-manager-metrics-service.` + operatorNS + `.svc:8443/metrics`

var _ = Describe("metrics endpoint", Label("e2e", "metrics"), func() {

	It("rejects unauthenticated request", func(ctx context.Context) {
		// TODO: remove -k once metrics service has serving-cert annotation and kube-rbac-proxy uses OCP service-CA cert
		logs, err := probe.RunPod(ctx, `curl -k --cacert /etc/cabundle/service-ca.crt `+metricsURL+` 2>&1`)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(logs)).To(ContainSubstring("Unauthorized"))
	})

	// TODO: re-enable once kube-rbac-proxy is removed and metrics serve directly over HTTPS
	XIt("allows request with valid SA token", func(ctx context.Context) {
		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "e2e-metrics-reader", Labels: e2eLabels},
			Rules: []rbacv1.PolicyRule{{
				NonResourceURLs: []string{"/metrics"},
				Verbs:           []string{"get"},
			}},
		}
		Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, cr))).To(Succeed())

		sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
			Name: "e2e-metrics-reader", Namespace: operatorNS, Labels: e2eLabels,
		}}
		Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, sa))).To(Succeed())

		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "e2e-metrics-reader", Labels: e2eLabels},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: operatorNS,
			}},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     cr.Name,
			},
		}
		Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, crb))).To(Succeed())

		tr, err := clientset.CoreV1().ServiceAccounts(operatorNS).
			CreateToken(ctx, sa.Name, &authv1.TokenRequest{
				Spec: authv1.TokenRequestSpec{ExpirationSeconds: ptr.To(int64(600))},
			}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// TODO: remove -k once metrics service has serving-cert annotation and kube-rbac-proxy uses OCP service-CA cert
		cmd := `curl -k --cacert /etc/cabundle/service-ca.crt -H "Authorization: Bearer ` + tr.Status.Token + `" ` + metricsURL + ` 2>&1`
		logs, err := probe.RunPod(ctx, cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(logs)).To(ContainSubstring("go_goroutines"))
	})

	XIt("metrics are scraped by Prometheus", func(_ context.Context) {
		// TODO: ServiceMonitor is not included in the OLM bundle manifests so
		// Prometheus does not scrape the operator when deployed via OLM.
		// Add ServiceMonitor to bundle/manifests before implementing this test.
	})
})
