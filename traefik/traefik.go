package traefik

import (
	"fmt"
	"strings"

	"github.com/zhiminwen/quote"
)

// example:
//

func ConstructHTTPSRoutersToml(hosts []string, uniqName string, lbAddress string) string {
	var domainList []string
	for _, host := range hosts {
		domainList = append(domainList, fmt.Sprintf("`%s`", host))
	}

	content := quote.Template(quote.HereDoc(`
		[tcp.routers]
		[tcp.routers.{{ .uniqName }}]
			entryPoints = ["https"]
			rule = "HostSNI({{ .hostList }})"
			service = "service-{{ .uniqName }}"
			[tcp.routers.{{ .uniqName }}.tls]
			passthrough = true

		[tcp.services]
		[tcp.services.service-{{ .uniqName }}.loadBalancer]
		[[tcp.services.service-{{ .uniqName }}.loadBalancer.servers]]
			address = "{{ .lbAddress }}"
	`), map[string]string{
		"quote":    "`",
		"cluster":  conf.Cluster,
		"domain":   conf.Domain,
		"lbIp":     conf.LoadBalancerIp,
		"hostList": strings.Join(domainList, ","),
	})
	master.Put(content, "/etc/traefik/conf.d/"+conf.Cluster+".toml")
}
