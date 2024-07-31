package validation

import (
	"fmt"
	"reflect"

	"github.com/mlguerrero12/pf-status-relay-operator/api/v1alpha1"
)

// InterfaceUniqueness validates that interfaces do not overlap for daemon sets that share nodes.
func InterfaceUniqueness(pfMonitor *v1alpha1.PFLACPMonitor, pfMonitorList *v1alpha1.PFLACPMonitorList) error {
	for _, monitor := range pfMonitorList.Items {
		if pfMonitor.Name == monitor.Name {
			continue
		}

		if monitor.Status.Degraded {
			continue
		}

		if monitor.Spec.NodeSelector == nil || pfMonitor.Spec.NodeSelector == nil || nodeSelectorOverlaps(pfMonitor.Spec.NodeSelector, monitor.Spec.NodeSelector) {
			if !areInterfacesUnique(pfMonitor.Spec.Interfaces, monitor.Spec.Interfaces) {
				return fmt.Errorf("interfaces %s conflict with the ones from PFLACPMonitor %s", pfMonitor.Spec.Interfaces, monitor.Name)
			}
		}
	}

	return nil
}

func nodeSelectorOverlaps(nodeSelector1, nodeSelector2 map[string]string) bool {
	for key, value := range nodeSelector1 {
		if v, ok := nodeSelector2[key]; ok && reflect.DeepEqual(value, v) {
			return true
		}
	}
	return false
}

func areInterfacesUnique(intfs1, intfs2 []string) bool {
	seen := make(map[string]struct{})
	for _, item := range intfs2 {
		seen[item] = struct{}{}
	}

	for _, item := range intfs1 {
		if _, ok := seen[item]; ok {
			return false
		}
	}
	return true
}
