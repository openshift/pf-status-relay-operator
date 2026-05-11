//go:build e2e

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
	"fmt"
	"os"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	pfstatusrelayv1alpha1 "github.com/openshift/pf-status-relay-operator/api/v1alpha1"
	"github.com/openshift/pf-status-relay-operator/e2e/pkg/prober"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const operatorNS = "openshift-pf-status-relay-operator"

var e2eLabels = map[string]string{"app.kubernetes.io/created-by": "e2e"}

var (
	k8sClient client.Client
	clientset kubernetes.Interface
	probe     *prober.Prober
)

var _ = BeforeSuite(func() {
	ctx := context.Background()
	e2eLabels["ginkgo-seed"] = strconv.FormatInt(GinkgoRandomSeed(), 10)

	cfg, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	Expect(err).NotTo(HaveOccurred())

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(configv1.Install(scheme))
	utilruntime.Must(pfstatusrelayv1alpha1.AddToScheme(scheme))

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	clientset, err = kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())

	By("verifying operator is deployed")
	ns := &corev1.Namespace{}
	err = k8sClient.Get(ctx, types.NamespacedName{Name: operatorNS}, ns)
	if apierrors.IsNotFound(err) {
		Fail("operator not deployed — deploy the operator before running e2e tests")
	}
	Expect(err).NotTo(HaveOccurred())

	deploy := &appsv1.Deployment{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      "pf-status-relay-operator-controller-manager",
		Namespace: operatorNS,
	}, deploy)
	if apierrors.IsNotFound(err) {
		Fail("operator deployment not found — deploy the operator before running e2e tests")
	}
	Expect(err).NotTo(HaveOccurred())
	_, _ = fmt.Fprintf(GinkgoWriter, "operator image: %s\n", deploy.Spec.Template.Spec.Containers[0].Image)

	probe = &prober.Prober{Client: k8sClient, Clientset: clientset, Namespace: operatorNS, Labels: e2eLabels}
})

var _ = AfterEach(func(ctx context.Context) {
	if CurrentSpecReport().Failed() {
		return
	}
	seedLabel := client.MatchingLabels{"ginkgo-seed": e2eLabels["ginkgo-seed"]}
	_ = k8sClient.DeleteAllOf(ctx, &rbacv1.ClusterRoleBinding{}, seedLabel)
	_ = k8sClient.DeleteAllOf(ctx, &rbacv1.ClusterRole{}, seedLabel)
	_ = k8sClient.DeleteAllOf(ctx, &corev1.ServiceAccount{}, client.InNamespace(operatorNS), seedLabel)
	_ = k8sClient.DeleteAllOf(ctx, &corev1.ConfigMap{}, client.InNamespace(operatorNS), seedLabel)
	_ = k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace(operatorNS), seedLabel)
})

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "e2e suite")
}
