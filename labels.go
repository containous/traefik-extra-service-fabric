package servicefabric

import (
	"github.com/containous/traefik/provider/label"
)

// Must be replace by:
// func hasFunc(labelName string) func(service ServiceItemExtended) bool {
// 	return func(service ServiceItemExtended) bool {
// 		return label.Has(service.Labels, labelName)
// 	}
// }
// Deprecated
func hasFunc() func(service ServiceItemExtended, labelName string) bool {
	return func(service ServiceItemExtended, labelName string) bool {
		return label.Has(service.Labels, labelName)
	}
}

func getFuncBoolLabel(labelName string, defaultValue bool) func(service ServiceItemExtended) bool {
	return func(service ServiceItemExtended) bool {
		return label.GetBoolValue(service.Labels, labelName, defaultValue)
	}
}

// Must be replace by:
// func getFuncStringLabel(labelName string, defaultValue string) func(service ServiceItemExtended) string {
// 	return func(service ServiceItemExtended) string {
// 		return label.GetStringValue(service.Labels, labelName, defaultValue)
// 	}
// }
// Deprecated
func getFuncStringLabel(defaultValue string) func(service ServiceItemExtended, labelName string) string {
	return func(service ServiceItemExtended, labelName string) string {
		return label.GetStringValue(service.Labels, labelName, defaultValue)
	}
}

// Must be replace by:
// func getFuncStringLabel(labelName string, defaultValue string) func(service ServiceItemExtended) string {
// 	return func(service ServiceItemExtended) string {
// 		return label.GetStringValue(service.Labels, labelName, defaultValue)
// 	}
// }
// Deprecated
func getFuncStringLabelWithDefault() func(service ServiceItemExtended, labelName string, defaultValue string) string {
	return func(service ServiceItemExtended, labelName string, defaultValue string) string {
		return label.GetStringValue(service.Labels, labelName, defaultValue)
	}
}
