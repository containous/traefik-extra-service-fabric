package servicefabric

const tmpl = `
{{$groupedServiceMap := getServices .Services "backend.group.name"}}
[backends]
    {{range $aggName, $aggServices := $groupedServiceMap }}
      [backends."{{$aggName}}"]
      {{range $service := $aggServices}}
        {{range $partition := $service.Partitions}}
          {{range $instance := $partition.Instances}}
            [backends."{{$aggName}}".servers."{{$service.ID}}-{{$instance.ID}}"]
            url = "{{getDefaultEndpoint $instance}}"
            weight = {{getLabelValue $service "backend.group.weight" "1"}}
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
        {{if hasLabel $service "backend.loadbalancer.method"}}
          method = "{{getLabelValue $service "backend.loadbalancer.method" "" }}"
        {{else}}
          method = "drr"
        {{end}}

        {{if hasLabel $service "backend.healthcheck.path"}}
          [backends."{{$service.Name}}".healthcheck]
          path = "{{getLabelValue $service "backend.healthcheck.path" ""}}"
          interval = "{{getLabelValue $service "backend.healthcheck.interval" "10s"}}"
        {{end}}

        {{if hasLabel $service "backend.loadbalancer.stickiness"}}
          [backends."{{$service.Name}}".LoadBalancer.stickiness]
        {{end}}

        {{if hasLabel $service "backend.circuitbreaker"}}
          [backends."{{$service.Name}}".circuitbreaker]
          expression = "{{getLabelValue $service "backend.circuitbreaker" ""}}"
        {{end}}

        {{if hasLabel $service "backend.maxconn.amount"}}
          [backends."{{$service.Name}}".maxconn]
          amount = {{getLabelValue $service "backend.maxconn.amount" ""}}
          {{if hasLabel $service "backend.maxconn.extractorfunc"}}
          extractorfunc = "{{getLabelValue $service "backend.maxconn.extractorfunc" ""}}"
          {{end}}
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
{{range $groupName, $groupServices := $groupedServiceMap}}
  {{$service := index $groupServices 0}}
    [frontends."{{$groupName}}"]
    backend = "{{$groupName}}"

    priority = {{ getPriority $service }}

    {{range $key, $value := getLabelsWithPrefix $service "frontend.rule"}}
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

    {{if hasLabel $service "frontend.whitelistSourceRange"}}
      whitelistSourceRange = {{getLabelValue $service "frontend.whitelistSourceRange"  ""}}
    {{end}}

    priority = {{ getPriority $service }}

    {{if hasLabel $service "frontend.auth.basic"}}
      basicAuth = {{getLabelValue $service "frontend.auth.basic" ""}}
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

    {{range $key, $value := getLabelsWithPrefix $service "frontend.rule"}}
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
