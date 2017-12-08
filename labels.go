package servicefabric

import (
	"strconv"
)

func getFuncBoolLabel(labelName string, defaultValue bool) func(service ServiceItemExtended) bool {
	return func(service ServiceItemExtended) bool {
		return getBoolValue(service.Labels, labelName, defaultValue)
	}
}

// ------

// Deprecated
func getFuncStringLabelWithDefault() func(service ServiceItemExtended, labelName string, defaultValue string) string {
	return func(service ServiceItemExtended, labelName string, defaultValue string) string {
		return getStringValue(service.Labels, labelName, defaultValue)
	}
}

// Deprecated
func hasFunc() func(service ServiceItemExtended, labelName string) bool {
	return func(service ServiceItemExtended, labelName string) bool {
		return hasLabelNew(service.Labels, labelName)
	}
}

// Deprecated
func getFuncStringLabel(defaultValue string) func(service ServiceItemExtended, labelName string) string {
	return func(service ServiceItemExtended, labelName string) string {
		return getStringValue(service.Labels, labelName, defaultValue)
	}
}

// must be replace by label.Has()
// Deprecated
func hasLabelNew(labels map[string]string, labelName string) bool {
	value, ok := labels[labelName]
	return ok && len(value) > 0
}

// must be replace by label.GetStringValue()
// Deprecated
func getStringValue(labels map[string]string, labelName string, defaultValue string) string {
	if value, ok := labels[labelName]; ok && len(value) > 0 {
		return value
	}
	return defaultValue
}

// must be replace by label.GetBoolValue()
// Deprecated
func getBoolValue(labels map[string]string, labelName string, defaultValue bool) bool {
	rawValue, ok := labels[labelName]
	if ok {
		v, err := strconv.ParseBool(rawValue)
		if err == nil {
			return v
		}
	}
	return defaultValue
}
