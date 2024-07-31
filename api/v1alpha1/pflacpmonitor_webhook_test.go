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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PFLACPMonitor Webhook", func() {

	Context("When creating PFLACPMonitor under Validating Webhook", func() {
		It("Should admit", func() {
			pfMonitor := &PFLACPMonitor{
				Spec: PFLACPMonitorSpec{
					Interfaces: []string{
						"eth0",
						"eth1",
					},
				},
			}

			_, err := pfMonitor.ValidateCreate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should deny if a required field is empty", func() {
			pfMonitor := &PFLACPMonitor{
				Spec: PFLACPMonitorSpec{
					Interfaces: []string{
						"eth0",
						"   ",
					},
				},
			}

			_, err := pfMonitor.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("Should deny if interfaces are not unique", func() {
			pfMonitor := &PFLACPMonitor{
				Spec: PFLACPMonitorSpec{
					Interfaces: []string{
						"eth0",
						"eth1",
						"eth0",
					},
				},
			}

			_, err := pfMonitor.ValidateUpdate(nil)
			Expect(err).To(HaveOccurred())
		})
	})

})
