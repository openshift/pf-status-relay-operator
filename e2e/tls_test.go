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
	"encoding/base64"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const webhookServiceName = "pf-status-relay-operator-controller-manager-service"

var _ = Describe("TLS compliance", Label("e2e", "tls"), func() {

	It("operator watches the cluster TLS profile", func(ctx context.Context) {
		pods := &corev1.PodList{}
		Expect(k8sClient.List(ctx, pods,
			client.InNamespace(operatorNS),
			client.MatchingLabels{"control-plane": "controller-manager"},
		)).To(Succeed())
		Expect(pods.Items).NotTo(BeEmpty())

		req := clientset.CoreV1().Pods(operatorNS).GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{})
		logs, err := req.DoRaw(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(logs)).To(ContainSubstring("tlssecurityprofilewatcher"),
			"operator should run the TLS security profile watcher")
	})

	DescribeTable("Modern profile",
		func(ctx context.Context, url string, webhook bool) {
			apiserver := &configv1.APIServer{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "cluster"}, apiserver)).To(Succeed())
			if apiserver.Spec.TLSSecurityProfile == nil ||
				apiserver.Spec.TLSSecurityProfile.Type != configv1.TLSProfileModernType {
				Skip("cluster TLS profile is not Modern")
			}
			caFile := "/etc/cabundle/service-ca.crt"
			caSetup := ""
			if webhook {
				caFile = "/tmp/webhook-ca.crt"
				caSetup = "printf %s " + base64.StdEncoding.EncodeToString(webhookCABundle(ctx)) +
					" | base64 -d > " + caFile + " && "
			}

			By("rejecting TLS 1.2")
			logs, err := probe.RunPod(ctx,
				caSetup+`curl --tlsv1.2 --tls-max 1.2 -sv -o /dev/null --cacert `+caFile+` `+url)
			Expect(err).To(HaveOccurred(), "expected server to reject TLS 1.2:\n%s", logs)

			By("accepting TLS 1.3")
			logs, err = probe.RunPod(ctx,
				caSetup+`curl --tlsv1.3 --tls-max 1.3 -sv -o /dev/null --cacert `+caFile+` `+url)
			Expect(err).NotTo(HaveOccurred(), "TLS 1.3 connection failed:\n%s", logs)

			By("rejecting P-521, which is outside the Modern profile")
			logs, err = probe.RunPod(ctx,
				caSetup+`curl --tlsv1.3 --tls-max 1.3 --curves P-521 -sv -o /dev/null --cacert `+caFile+` `+url)
			Expect(err).To(HaveOccurred(), "expected server to reject P-521-only client:\n%s", logs)
			Expect(string(logs)).To(ContainSubstring("handshake failure"), "expected a TLS handshake rejection")

			By("accepting P-384, which is in the Modern profile")
			logs, err = probe.RunPod(ctx,
				caSetup+`curl --tlsv1.3 --tls-max 1.3 --curves P-384 -sv -o /dev/null --cacert `+caFile+` `+url)
			Expect(err).NotTo(HaveOccurred(), "expected P-384 connection to succeed:\n%s", logs)
		},
		Entry("metrics endpoint", Label("metrics"),
			`https://pf-status-relay-operator-controller-manager-metrics-service.`+operatorNS+`.svc:8443/metrics`, false),
		Entry("webhook endpoint", Label("webhook"),
			`https://pf-status-relay-operator-controller-manager-service.`+operatorNS+`.svc:443/`, true),
	)
})

func webhookCABundle(ctx context.Context) []byte {
	GinkgoHelper()
	configs, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())

	for _, config := range configs.Items {
		for _, hook := range config.Webhooks {
			svc := hook.ClientConfig.Service
			if svc != nil && svc.Namespace == operatorNS && svc.Name == webhookServiceName {
				Expect(hook.ClientConfig.CABundle).NotTo(BeEmpty(), "webhook CA bundle is empty")
				return hook.ClientConfig.CABundle
			}
		}
	}
	Fail("validating webhook for service " + webhookServiceName + "." + operatorNS + " not found")
	return nil
}
