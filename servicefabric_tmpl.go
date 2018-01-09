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

            {{$backendName := getBackendName $service.Name $partition}}
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
    
    [frontends."frontend-{{$frontend}}".headers]
    {{if hasFrameDenyHeaders $service}}
    FrameDeny = {{getFrameDenyHeaders $service}}
    {{end}}

    {{if hasRequestHeaders $service}}
      [frontends."frontend-{{$frontend}}".headers.customrequestheaders]
      {{range $k, $v := getRequestHeaders $service}}
      {{$k}} = "{{$v}}"
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
