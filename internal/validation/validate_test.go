package validation

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/pf-status-relay-operator/api/v1alpha1"
)

var _ = Describe("Validator", func() {
	Describe("NodeSelector", func() {
		var (
			pfMonitor1, pfMonitor2 *v1alpha1.PFLACPMonitor
			pfMonitorList          *v1alpha1.PFLACPMonitorList
		)

		BeforeEach(func() {
			pfMonitor1 = &v1alpha1.PFLACPMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name: "monitor1",
				},
			}

			pfMonitor2 = &v1alpha1.PFLACPMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name: "monitor2",
				},
			}
		})

		It("should pass if interfaces are equal but node selectors different", func() {
			pfMonitor1.Spec.NodeSelector = map[string]string{"key1": "value1"}
			pfMonitor1.Spec.Interfaces = []string{"eth0"}

			pfMonitor2.Spec.NodeSelector = map[string]string{"key2": "value2"}
			pfMonitor2.Spec.Interfaces = []string{"eth0"}

			pfMonitorList = &v1alpha1.PFLACPMonitorList{
				Items: []v1alpha1.PFLACPMonitor{*pfMonitor1, *pfMonitor2},
			}

			err := InterfaceUniqueness(pfMonitor1, pfMonitorList)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error if interfaces are equal and node selector overlaps", func() {
			pfMonitor1.Spec.NodeSelector = map[string]string{"key1": "value1", "key2": "value2"}
			pfMonitor1.Spec.Interfaces = []string{"eth0", "eth3"}

			pfMonitor2.Spec.NodeSelector = map[string]string{"key2": "value2"}
			pfMonitor2.Spec.Interfaces = []string{"eth2", "eth3"}

			pfMonitorList = &v1alpha1.PFLACPMonitorList{
				Items: []v1alpha1.PFLACPMonitor{*pfMonitor1, *pfMonitor2},
			}

			err := InterfaceUniqueness(pfMonitor1, pfMonitorList)
			Expect(err).To(HaveOccurred())
		})

		It("should return an error when NodeSelector is nil and interfaces are equal - new PFLACPMonitor", func() {
			pfMonitor1.Spec.NodeSelector = nil
			pfMonitor1.Spec.Interfaces = []string{"eth0"}

			pfMonitor2.Spec.NodeSelector = map[string]string{"key2": "value2"}
			pfMonitor2.Spec.Interfaces = []string{"eth0"}

			pfMonitorList = &v1alpha1.PFLACPMonitorList{
				Items: []v1alpha1.PFLACPMonitor{*pfMonitor1, *pfMonitor2},
			}

			err := InterfaceUniqueness(pfMonitor1, pfMonitorList)
			Expect(err).To(HaveOccurred())
		})

		It("should pass when NodeSelector is nil and interfaces are different - new PFLACPMonitor", func() {
			pfMonitor1.Spec.NodeSelector = nil
			pfMonitor1.Spec.Interfaces = []string{"eth0"}

			pfMonitor2.Spec.NodeSelector = map[string]string{"key2": "value2"}
			pfMonitor2.Spec.Interfaces = []string{"eth1"}

			pfMonitorList = &v1alpha1.PFLACPMonitorList{
				Items: []v1alpha1.PFLACPMonitor{*pfMonitor1, *pfMonitor2},
			}

			err := InterfaceUniqueness(pfMonitor1, pfMonitorList)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error when NodeSelector is nil and interfaces are equal - existing PFLACPMonitor", func() {
			pfMonitor1.Spec.NodeSelector = map[string]string{"key1": "value1"}
			pfMonitor1.Spec.Interfaces = []string{"eth2", "eth3"}

			pfMonitor2.Spec.NodeSelector = nil
			pfMonitor2.Spec.Interfaces = []string{"eth0", "eth3"}

			pfMonitorList = &v1alpha1.PFLACPMonitorList{
				Items: []v1alpha1.PFLACPMonitor{*pfMonitor1, *pfMonitor2},
			}

			err := InterfaceUniqueness(pfMonitor1, pfMonitorList)
			Expect(err).To(HaveOccurred())
		})

		It("should pass when NodeSelector is nil and interfaces are different - existing PFLACPMonitor", func() {
			pfMonitor1.Spec.NodeSelector = map[string]string{"key1": "value1"}
			pfMonitor1.Spec.Interfaces = []string{"eth0"}

			pfMonitor2.Spec.NodeSelector = nil
			pfMonitor2.Spec.Interfaces = []string{"eth1", "eth2"}

			pfMonitorList = &v1alpha1.PFLACPMonitorList{
				Items: []v1alpha1.PFLACPMonitor{*pfMonitor1, *pfMonitor2},
			}

			err := InterfaceUniqueness(pfMonitor1, pfMonitorList)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
