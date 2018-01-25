package servicefabric

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"text/template"

	"github.com/containous/traefik/log"
	"github.com/containous/traefik/provider"
	"github.com/containous/traefik/provider/label"
	"github.com/containous/traefik/types"
	sf "github.com/jjcollinge/servicefabric"
)

func (p *Provider) buildConfiguration(sfClient sfClient) (*types.Configuration, error) {
	var sfFuncMap = template.FuncMap{

		// Services
		"getServices":                getServices,
		"hasLabel":                   hasService,
		"getLabelValue":              getServiceStringLabel,
		"getLabelsWithPrefix":        getServiceLabelsWithPrefix,
		"isPrimary":                  isPrimary,
		"isStateful":                 isStateful,
		"isStateless":                isStateless,
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

		// Headers
		"hasHeaders":                        hasHeaders,
		"hasRequestHeaders":                 hasFuncService(label.TraefikFrontendRequestHeaders),
		"getRequestHeaders":                 getFuncMapLabel(label.TraefikFrontendRequestHeaders),
		"hasResponseHeaders":                hasFuncService(label.TraefikFrontendResponseHeaders),
		"getResponseHeaders":                getFuncMapLabel(label.TraefikFrontendResponseHeaders),
		"hasAllowedHostsHeaders":            hasFuncService(label.TraefikFrontendAllowedHosts),
		"getAllowedHostsHeaders":            getFuncServiceSliceStringLabel(label.TraefikFrontendAllowedHosts),
		"hasHostsProxyHeaders":              hasFuncService(label.TraefikFrontendHostsProxyHeaders),
		"getHostsProxyHeaders":              getFuncServiceSliceStringLabel(label.TraefikFrontendHostsProxyHeaders),
		"hasSSLRedirectHeaders":             hasFuncService(label.TraefikFrontendSSLRedirect),
		"getSSLRedirectHeaders":             getFuncBoolLabel(label.TraefikFrontendSSLRedirect, false),
		"hasSSLTemporaryRedirectHeaders":    hasFuncService(label.TraefikFrontendSSLTemporaryRedirect),
		"getSSLTemporaryRedirectHeaders":    getFuncBoolLabel(label.TraefikFrontendSSLTemporaryRedirect, false),
		"hasSSLHostHeaders":                 hasFuncService(label.TraefikFrontendSSLHost),
		"getSSLHostHeaders":                 getFuncServiceSliceStringLabel(label.TraefikFrontendSSLHost),
		"hasSSLProxyHeaders":                hasFuncService(label.TraefikFrontendSSLProxyHeaders),
		"getSSLProxyHeaders":                getFuncMapLabel(label.TraefikFrontendSSLProxyHeaders),
		"hasSTSSecondsHeaders":              hasFuncService(label.TraefikFrontendSTSSeconds),
		"getSTSSecondsHeaders":              getFuncInt64Label(label.TraefikFrontendSTSSeconds, 0),
		"hasSTSIncludeSubdomainsHeaders":    hasFuncService(label.TraefikFrontendSTSIncludeSubdomains),
		"getSTSIncludeSubdomainsHeaders":    getFuncBoolLabel(label.TraefikFrontendSTSIncludeSubdomains, false),
		"hasSTSPreloadHeaders":              hasFuncService(label.TraefikFrontendSTSPreload),
		"getSTSPreloadHeaders":              getFuncBoolLabel(label.TraefikFrontendSTSPreload, false),
		"hasForceSTSHeaderHeaders":          hasFuncService(label.TraefikFrontendForceSTSHeader),
		"getForceSTSHeaderHeaders":          getFuncBoolLabel(label.TraefikFrontendForceSTSHeader, false),
		"hasFrameDenyHeaders":               hasFuncService(label.TraefikFrontendFrameDeny),
		"getFrameDenyHeaders":               getFuncBoolLabel(label.TraefikFrontendFrameDeny, false),
		"hasCustomFrameOptionsValueHeaders": hasFuncService(label.TraefikFrontendCustomFrameOptionsValue),
		"getCustomFrameOptionsValueHeaders": getFuncServiceSliceStringLabel(label.TraefikFrontendCustomFrameOptionsValue),
		"hasContentTypeNosniffHeaders":      hasFuncService(label.TraefikFrontendContentTypeNosniff),
		"getContentTypeNosniffHeaders":      getFuncBoolLabel(label.TraefikFrontendContentTypeNosniff, false),
		"hasBrowserXSSFilterHeaders":        hasFuncService(label.TraefikFrontendBrowserXSSFilter),
		"getBrowserXSSFilterHeaders":        getFuncBoolLabel(label.TraefikFrontendBrowserXSSFilter, false),
		"hasContentSecurityPolicyHeaders":   hasFuncService(label.TraefikFrontendContentSecurityPolicy),
		"getContentSecurityPolicyHeaders":   getFuncServiceSliceStringLabel(label.TraefikFrontendContentSecurityPolicy),
		"hasPublicKeyHeaders":               hasFuncService(label.TraefikFrontendPublicKey),
		"getPublicKeyHeaders":               getFuncServiceSliceStringLabel(label.TraefikFrontendPublicKey),
		"hasReferrerPolicyHeaders":          hasFuncService(label.TraefikFrontendReferrerPolicy),
		"getReferrerPolicyHeaders":          getFuncServiceSliceStringLabel(label.TraefikFrontendReferrerPolicy),
		"hasIsDevelopmentHeaders":           hasFuncService(label.TraefikFrontendIsDevelopment),
		"getIsDevelopmentHeaders":           getFuncBoolLabel(label.TraefikFrontendIsDevelopment, false),

		// SF Service Grouping
		"getGroupedServices": getFuncServicesGroupedByLabel(traefikSFGroupName),
		"getGroupedWeight":   getFuncServiceStringLabel(traefikSFGroupWeight, "1"),
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

func isStateful(service ServiceItemExtended) bool {
	return service.ServiceKind == "Stateful"
}

func isStateless(service ServiceItemExtended) bool {
	return service.ServiceKind == "Stateless"
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

func hasHeaders(service ServiceItemExtended) bool {
	for key := range service.Labels {
		if strings.HasPrefix(key, label.TraefikFrontendHeaders) {
			return true
		}
	}
	return false
}

func getBackendName(service ServiceItemExtended, partition PartitionItemExtended) string {
	return provider.Normalize(service.Name + partition.PartitionInformation.ID)
}

func getDefaultEndpoint(instance replicaInstance) string {
	id, data := instance.GetReplicaData()
	endpoint, err := getReplicaDefaultEndpoint(data)
	if err != nil {
		log.Warnf("No default endpoint for replica %s in service %s endpointData: %s", id, data.Address)
		return ""
	}
	return endpoint
}

func getReplicaDefaultEndpoint(replicaData *sf.ReplicaItemBase) (string, error) {
	endpoints, err := decodeEndpointData(replicaData.Address)
	if err != nil {
		return "", err
	}

	var defaultHTTPEndpoint string
	for _, v := range endpoints {
		if strings.Contains(v, "http") {
			defaultHTTPEndpoint = v
			break
		}
	}

	if len(defaultHTTPEndpoint) == 0 {
		return "", errors.New("no default endpoint found")
	}
	return defaultHTTPEndpoint, nil
}

func getNamedEndpoint(instance replicaInstance, endpointName string) string {
	id, data := instance.GetReplicaData()
	endpoint, err := getReplicaNamedEndpoint(data, endpointName)
	if err != nil {
		log.Warnf("No names endpoint of %s for replica %s in endpointData: %s. Error: %v", endpointName, id, data.Address, err)
		return ""
	}
	return endpoint
}

func getReplicaNamedEndpoint(replicaData *sf.ReplicaItemBase, endpointName string) (string, error) {
	endpoints, err := decodeEndpointData(replicaData.Address)
	if err != nil {
		return "", err
	}

	endpoint, exists := endpoints[endpointName]
	if !exists {
		return "", errors.New("endpoint doesn't exist")
	}
	return endpoint, nil
}

func getApplicationParameter(app sf.ApplicationItem, key string) string {
	for _, param := range app.Parameters {
		if param.Key == key {
			return param.Value
		}
	}
	log.Errorf("Parameter %s doesn't exist in app %s", key, app.Name)
	return ""
}

func getServices(services []ServiceItemExtended, key string) map[string][]ServiceItemExtended {
	result := map[string][]ServiceItemExtended{}
	for _, service := range services {
		if value, exists := service.Labels[key]; exists {
			if matchingServices, hasKeyAlready := result[value]; hasKeyAlready {
				result[value] = append(matchingServices, service)
			} else {
				result[value] = []ServiceItemExtended{service}
			}
		}
	}
	return result
}

func doesAppParamContain(app sf.ApplicationItem, key, shouldContain string) bool {
	value := getApplicationParameter(app, key)
	return strings.Contains(value, shouldContain)
}

func filterServicesByLabelValue(services []ServiceItemExtended, key, expectedValue string) []ServiceItemExtended {
	var srvWithLabel []ServiceItemExtended
	for _, service := range services {
		value, exists := service.Labels[key]
		if exists && value == expectedValue {
			srvWithLabel = append(srvWithLabel, service)
		}
	}
	return srvWithLabel
}

func decodeEndpointData(endpointData string) (map[string]string, error) {
	var endpointsMap map[string]map[string]string

	if endpointData == "" {
		return nil, errors.New("endpoint data is empty")
	}

	err := json.Unmarshal([]byte(endpointData), &endpointsMap)
	if err != nil {
		return nil, err
	}

	endpoints, endpointsExist := endpointsMap["Endpoints"]
	if !endpointsExist {
		return nil, errors.New("endpoint doesn't exist in endpoint data")
	}

	return endpoints, nil
}
