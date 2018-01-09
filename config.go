package servicefabric

import (
	"math"
	"text/template"

	"github.com/containous/traefik/provider/label"
	"github.com/containous/traefik/types"
)

func (p *Provider) buildConfiguration(sfClient sfClient) (*types.Configuration, error) {
	var sfFuncMap = template.FuncMap{

		// Services
		"getServices":                getServices,
		"hasLabel":                   hasService,
		"getLabelValue":              getServiceStringLabel,
		"getLabelsWithPrefix":        getServiceLabelsWithPrefix,
		"isPrimary":                  isPrimary,
		"isEnabled":                  getFuncBoolLabel(label.TraefikEnable, false),
		"getBackendName":             getBackendName,
		"getDefaultEndpoint":         getDefaultEndpoint,
		"getNamedEndpoint":           getNamedEndpoint,           // FIXME unused
		"getApplicationParameter":    getApplicationParameter,    // FIXME unused
		"doesAppParamContain":        doesAppParamContain,        // FIXME unused
		"filterServicesByLabelValue": filterServicesByLabelValue, // FIXME unused

		// Backend functions
		"getWeight":                   getFuncServiceStringLabel(label.TraefikWeight, label.DefaultWeight),
		"getProtocol":                 getFuncServiceStringLabel(label.TraefikProtocol, label.DefaultProtocol),
		"hasHealthCheckLabels":        hasFuncService(label.TraefikBackendHealthCheckPath),
		"getHealthCheckPath":          getFuncServiceStringLabel(label.TraefikBackendHealthCheckPath, ""),
		"getHealthCheckPort":          getFuncServiceStringLabel(label.TraefikBackendHealthCheckPort, "0"),
		"getHealthCheckInterval":      getFuncServiceStringLabel(label.TraefikBackendHealthCheckInterval, ""),
		"hasCircuitBreakerLabel":      hasFuncService(label.TraefikBackendCircuitBreakerExpression),
		"getCircuitBreakerExpression": getFuncServiceStringLabel(label.TraefikBackendCircuitBreakerExpression, label.DefaultCircuitBreakerExpression),
		"hasLoadBalancerLabel":        hasLoadBalancerLabel,
		"getLoadBalancerMethod":       getFuncServiceStringLabel(label.TraefikBackendLoadBalancerMethod, label.DefaultBackendLoadBalancerMethod),
		"hasMaxConnLabels":            hasMaxConnLabels,
		"getMaxConnAmount":            getFuncServiceStringLabel(label.TraefikBackendMaxConnAmount, string(math.MaxInt64)),
		"getMaxConnExtractorFunc":     getFuncServiceStringLabel(label.TraefikBackendMaxConnExtractorFunc, label.DefaultBackendMaxconnExtractorFunc),
		"getSticky":                   getFuncBoolLabel(label.TraefikBackendLoadBalancerSticky, false),
		"hasStickinessLabel":          hasFuncService(label.TraefikBackendLoadBalancerStickiness),
		"getStickinessCookieName":     getFuncServiceStringLabel(label.TraefikBackendLoadBalancerStickinessCookieName, label.DefaultBackendLoadbalancerStickinessCookieName),

		// Frontend Functions
		"getPriority":             getFuncServiceStringLabel(label.TraefikFrontendPriority, label.DefaultFrontendPriority),
		"hasRequestHeaders":       hasFuncService(label.TraefikFrontendRequestHeaders),
		"getRequestHeaders":       getFuncServiceMapLabel(label.TraefikFrontendRequestHeaders),
		"hasFrameDenyHeaders":     hasFuncService(label.TraefikFrontendFrameDeny),
		"getFrameDenyHeaders":     getFuncBoolLabel(label.TraefikFrontendFrameDeny, false),
		"getPassHostHeader":       getFuncServiceStringLabel(label.TraefikFrontendPassHostHeader, label.DefaultPassHostHeader),
		"getPassTLSCert":          getFuncBoolLabel(label.TraefikFrontendPassTLSCert, false),
		"hasEntryPoints":          hasFuncService(label.TraefikFrontendEntryPoints),
		"getEntryPoints":          getFuncServiceSliceStringLabel(label.TraefikFrontendEntryPoints),
		"hasBasicAuth":            hasFuncService(label.TraefikFrontendAuthBasic),
		"getBasicAuth":            getFuncServiceSliceStringLabel(label.TraefikFrontendAuthBasic),
		"getWhitelistSourceRange": getFuncServiceSliceStringLabel(label.TraefikFrontendWhitelistSourceRange),
		"hasRedirect":             hasRedirect,
		"getRedirectEntryPoint":   getFuncServiceStringLabel(label.TraefikFrontendRedirectEntryPoint, label.DefaultFrontendRedirectEntryPoint),
		"getRedirectRegex":        getFuncServiceStringLabel(label.TraefikFrontendRedirectRegex, ""),
		"getRedirectReplacement":  getFuncServiceStringLabel(label.TraefikFrontendRedirectReplacement, ""),
		"getFrontendRules":        getFuncServiceLabelWithPrefix(label.TraefikFrontendRule),

		// SF Service Grouping
		"getGroupedServices": getFuncServicesGroupedByLabel(TraefikSFGroupName),
		"getGroupedWeight":   getFuncServiceStringLabel(TraefikSFGroupWeight, "1"),
	}

	services, err := getClusterServices(sfClient)
	if err != nil {
		return nil, err
	}

	templateObjects := struct {
		Services []ServiceItemExtended
	}{
		Services: services,
	}

	return p.GetConfiguration(tmpl, sfFuncMap, templateObjects)
}

func hasRedirect(service ServiceItemExtended) bool {
	return label.Has(service.Labels, label.TraefikFrontendRedirectEntryPoint) ||
		label.Has(service.Labels, label.TraefikFrontendRedirectReplacement) && label.Has(service.Labels, label.TraefikFrontendRedirectRegex)
}

func hasLoadBalancerLabel(service ServiceItemExtended) bool {
	method := label.Has(service.Labels, label.TraefikBackendLoadBalancerMethod)
	sticky := label.Has(service.Labels, label.TraefikBackendLoadBalancerSticky)
	stickiness := label.Has(service.Labels, label.TraefikBackendLoadBalancerStickiness)
	cookieName := label.Has(service.Labels, label.TraefikBackendLoadBalancerStickinessCookieName)
	return method || sticky || stickiness || cookieName
}

func hasMaxConnLabels(service ServiceItemExtended) bool {
	mca := label.Has(service.Labels, label.TraefikBackendMaxConnAmount)
	mcef := label.Has(service.Labels, label.TraefikBackendMaxConnExtractorFunc)
	return mca && mcef
}
