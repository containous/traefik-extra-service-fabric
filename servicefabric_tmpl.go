package servicefabric

const tmpl = `
[backends]
    {{range $aggName, $aggServices := getGroupedServices .Services }}
      [backends."{{$aggName}}"]
      {{range $service := $aggServices}}
        {{range $partition := $service.Partitions}}
          {{range $instance := $partition.Instances}}
            [backends."{{$aggName}}".servers."{{$service.ID}}-{{$instance.ID}}"]
            url = "{{getDefaultEndpoint $instance}}"
            weight = {{getGroupedWeight $service}}
          {{end}}
        {{end}}
      {{end}}
    {{end}}
  {{range $service := .Services}}
  {{if isEnabled $service}}
    {{range $partition := $service.Partitions}}
      {{if eq $partition.ServiceKind "Stateless"}} 
      [backends."{{$service.Name}}"]
        [backends."{{$service.Name}}".LoadBalancer]
        {{if hasLoadBalancerLabel $service}}
          method = "{{getLoadBalancerMethod $service }}"
        {{end}}

        {{if hasHealthCheckLabels $service}}
          [backends."{{$service.Name}}".healthcheck]
          path = "{{getHealthCheckPath $service}}"
          interval = "{{getHealthCheckInterval $service }}"
          port = {{getHealthCheckPort $service}}
        {{end}}

        {{if hasStickinessLabel $service}}
          [backends."{{$service.Name}}".LoadBalancer.stickiness]
        {{end}}

        sticky = {{getSticky $service}}
        {{if hasStickinessLabel $service}}
        [backends."{{$service.Name}}".loadBalancer.stickiness]
          cookieName = "{{getStickinessCookieName $service}}"
        {{end}}

        {{if hasCircuitBreakerLabel $service}}
        [backends."{{$service.Name}}".circuitBreaker]
          expression = "{{getCircuitBreakerExpression $service}}"
        {{end}}

        {{if hasMaxConnLabels $service}}
        [backends."{{$service.Name}}".maxConn]
          amount = {{getMaxConnAmount $service}}
          extractorFunc = "{{getMaxConnExtractorFunc $service}}"
        {{end}}

        {{range $instance := $partition.Instances}}
          [backends."{{$service.Name}}".servers."{{$instance.ID}}"]
          url = "{{getDefaultEndpoint $instance}}"
          weight = {{getLabelValue $service "backend.weight" "1"}}
        {{end}}
      {{else if eq $partition.ServiceKind "Stateful"}}
        {{range $replica := $partition.Replicas}}
          {{if isPrimary $replica}}

            {{$backendName := getBackendName $service $partition}}
            [backends."{{$backendName}}".servers."{{$replica.ID}}"]
            url = "{{getDefaultEndpoint $replica}}"
            weight = 1

            [backends."{{$backendName}}".LoadBalancer]
            method = "drr"

            [backends."{{$backendName}}".circuitbreaker]
            expression = "NetworkErrorRatio() > 0.5"

          {{end}}
        {{end}}
      {{end}}
    {{end}}
  {{end}}
{{end}}

[frontends]
{{range $groupName, $groupServices := getGroupedServices .Services}}
  {{$service := index $groupServices 0}}
    [frontends."{{$groupName}}"]
    backend = "{{$groupName}}"

    priority = 50

    {{range $key, $value := getFrontendRules $service}}
    [frontends."{{$groupName}}".routes."{{$key}}"]
    rule = "{{$value}}"
    {{end}}
{{end}}
{{range $service := .Services}}
  {{if isEnabled $service}}
    {{$frontend := $service.Name}}
    {{if eq $service.ServiceKind "Stateless"}}
    
    [frontends."frontend-{{$frontend}}"]
    backend = "{{$service.Name}}"

    passHostHeader = {{getPassHostHeader $service }}

    passTLSCert = {{getPassTLSCert $service}}

    {{if getWhitelistSourceRange $service}}
    whitelistSourceRange = [{{range getWhitelistSourceRange $service}}
      "{{.}}",
      {{end}}]
    {{end}}

    priority = {{ getPriority $service }}

    {{if hasBasicAuth $service}}
      basicAuth = [{{range getBasicAuth $service }}
      "{{.}}",
      {{end}}]
    {{end}}

    {{if hasEntryPoints $service}}
    entryPoints = [{{range getEntryPoints $service}}
    "{{.}}",
    {{end}}]
    {{end}}
    
    {{ if hasHeaders $service}}
    [frontends."frontend-{{$frontend}}".headers]
      {{if hasSSLRedirectHeaders $service}}
      SSLRedirect = {{getSSLRedirectHeaders $service}}
      {{end}}
      {{if hasSSLTemporaryRedirectHeaders $service}}
      SSLTemporaryRedirect = {{getSSLTemporaryRedirectHeaders $service}}
      {{end}}
      {{if hasSSLHostHeaders $service}}
      SSLHost = "{{getSSLHostHeaders $service}}"
      {{end}}
      {{if hasSTSSecondsHeaders $service}}
      STSSeconds = {{getSTSSecondsHeaders $service}}
      {{end}}
      {{if hasSTSIncludeSubdomainsHeaders $service}}
      STSIncludeSubdomains = {{getSTSIncludeSubdomainsHeaders $service}}
      {{end}}
      {{if hasSTSPreloadHeaders $service}}
      STSPreload = {{getSTSPreloadHeaders $service}}
      {{end}}
      {{if hasForceSTSHeaderHeaders $service}}
      ForceSTSHeader = {{getForceSTSHeaderHeaders $service}}
      {{end}}
      {{if hasFrameDenyHeaders $service}}
      FrameDeny = {{getFrameDenyHeaders $service}}
      {{end}}
      {{if hasCustomFrameOptionsValueHeaders $service}}
      CustomFrameOptionsValue = "{{getCustomFrameOptionsValueHeaders $service}}"
      {{end}}
      {{if hasContentTypeNosniffHeaders $service}}
      ContentTypeNosniff = {{getContentTypeNosniffHeaders $service}}
      {{end}}
      {{if hasBrowserXSSFilterHeaders $service}}
      BrowserXSSFilter = {{getBrowserXSSFilterHeaders $service}}
      {{end}}
      {{if hasContentSecurityPolicyHeaders $service}}
      ContentSecurityPolicy = "{{getContentSecurityPolicyHeaders $service}}"
      {{end}}
      {{if hasPublicKeyHeaders $service}}
      PublicKey = "{{getPublicKeyHeaders $service}}"
      {{end}}
      {{if hasReferrerPolicyHeaders $service}}
      ReferrerPolicy = "{{getReferrerPolicyHeaders $service}}"
      {{end}}
      {{if hasIsDevelopmentHeaders $service}}
      IsDevelopment = {{getIsDevelopmentHeaders $service}}
      {{end}}

      {{if hasAllowedHostsHeaders $service}}
      AllowedHosts = [{{range getAllowedHostsHeaders $service}}
        "{{.}}",
        {{end}}]
      {{end}}

      {{if hasHostsProxyHeaders $service}}
      HostsProxyHeaders = [{{range getHostsProxyHeaders $service}}
        "{{.}}",
        {{end}}]
      {{end}}

      {{if hasRequestHeaders $service}}
        [frontends."frontend-{{$frontend}}".headers.customRequestHeaders]
        {{range $k, $v := getRequestHeaders $service}}
        {{$k}} = "{{$v}}"
        {{end}}
      {{end}}

      {{if hasResponseHeaders $service}}
      [frontends."frontend-{{$frontend}}".headers.customResponseHeaders]
        {{range $k, $v := getResponseHeaders $service}}
        {{$k}} = "{{$v}}"
        {{end}}
      {{end}}

      {{if hasSSLProxyHeaders $service}}
      [frontends."frontend-{{$frontend}}".headers.SSLProxyHeaders]
        {{range $k, $v := getSSLProxyHeaders $service}}
        {{$k}} = "{{$v}}"
        {{end}}
      {{end}}
    {{end}}

    {{range $key, $value := getFrontendRules $service}}
    [frontends."frontend-{{$frontend}}".routes."{{$key}}"]
    rule = "{{$value}}"
    {{end}}

    {{else if eq $service.ServiceKind "Stateful"}}
      {{range $partition := $service.Partitions}}
        {{$partitionId := $partition.PartitionInformation.ID}}

        {{if hasLabel $service "frontend.rule"}}
          [frontends."{{$service.Name}}/{{$partitionId}}"]
          backend = "{{getBackendName $service.Name $partition}}"
          [frontends."{{$service.Name}}/{{$partitionId}}".routes.default]
          rule = {{getLabelValue $service "frontend.rule.partition.$partitionId" ""}}

      {{end}}
    {{end}}
  {{end}}
{{end}}
{{end}}
`
