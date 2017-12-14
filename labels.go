package servicefabric

import (
	"strings"

	"github.com/containous/traefik/provider/label"
)

func getFuncBoolLabel(labelName string, defaultValue bool) func(service ServiceItemExtended) bool {
	return func(service ServiceItemExtended) bool {
		return label.GetBoolValue(service.Labels, labelName, defaultValue)
	}
}

func getFuncServiceStringLabel(service ServiceItemExtended, labelName string, defaultValue string) string {
	return label.GetStringValue(service.Labels, labelName, defaultValue)
}

func hasFuncService(service ServiceItemExtended, labelName string) bool {
	return label.Has(service.Labels, labelName)
}

func getServiceLabelsWithPrefix(service ServiceItemExtended, prefix string) map[string]string {
	results := make(map[string]string)
	for k, v := range service.Labels {
		if strings.HasPrefix(k, prefix) {
			results[k] = v
		}
	}
	return results
}
