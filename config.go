package servicefabric

import (
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
		"getBasicAuth":            getFuncServiceSliceStringLabel(label.TraefikFrontendAuthBasic),
		"getWhitelistSourceRange": getFuncServiceSliceStringLabel(label.TraefikFrontendWhitelistSourceRange),
		"hasRedirect":             hasRedirect,
		"getRedirectEntryPoint":   getFuncServiceStringLabel(label.TraefikFrontendRedirectEntryPoint, label.DefaultFrontendRedirectEntryPoint),
		"getRedirectRegex":        getFuncServiceStringLabel(label.TraefikFrontendRedirectRegex, ""),
		"getRedirectReplacement":  getFuncServiceStringLabel(label.TraefikFrontendRedirectReplacement, ""),
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
