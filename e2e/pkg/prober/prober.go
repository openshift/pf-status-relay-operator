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

package prober

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Prober struct {
	Client    client.Client
	Clientset kubernetes.Interface
	Namespace string
	Labels    map[string]string
}

func (p *Prober) RunPod(ctx context.Context, cmd string) ([]byte, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-probe-cabundle",
			Namespace: p.Namespace,
			Labels:    p.Labels,
			Annotations: map[string]string{
				"service.beta.openshift.io/inject-cabundle": "true",
			},
		},
	}
	if err := client.IgnoreAlreadyExists(p.Client.Create(ctx, cm)); err != nil {
		return nil, err
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "e2e-exec-",
			Namespace:    p.Namespace,
			Labels:       p.Labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:    "exec",
				Image:   "registry.access.redhat.com/ubi9/ubi-minimal:latest",
				Command: []string{"sh", "-c", cmd},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "cabundle", MountPath: "/etc/cabundle"},
				},
			}},
			Volumes: []corev1.Volume{{
				Name: "cabundle",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "e2e-probe-cabundle"},
					},
				},
			}},
		},
	}
	if err := p.Client.Create(ctx, pod); err != nil {
		return nil, err
	}
	podName := pod.Name

	var podPhase corev1.PodPhase
	var logs []byte

	if err := wait.PollUntilContextTimeout(ctx, 2*time.Second, 60*time.Second, false,
		func(ctx context.Context) (bool, error) {
			got := &corev1.Pod{}
			if err := p.Client.Get(ctx, types.NamespacedName{Name: podName, Namespace: p.Namespace}, got); err != nil {
				return false, err
			}
			switch got.Status.Phase {
			case corev1.PodSucceeded, corev1.PodFailed:
				podPhase = got.Status.Phase
				var logErr error
				logs, logErr = p.Clientset.CoreV1().Pods(p.Namespace).
					GetLogs(podName, &corev1.PodLogOptions{}).Do(ctx).Raw()
				if logErr != nil {
					fmt.Fprintf(ginkgo.GinkgoWriter, "warning: failed to fetch logs for pod %s: %v\n", podName, logErr)
				}
				return true, nil
			}
			return false, nil
		},
	); err != nil {
		return nil, fmt.Errorf("pod %s did not finish within 60s: %w", podName, err)
	}
	if podPhase == corev1.PodFailed {
		return logs, fmt.Errorf("pod failed")
	}
	return logs, nil
}
