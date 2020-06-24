package ubuntu

import (
	"log"

	"github.com/zhiminwen/quote"
)

type NetworkSpec struct {
	Address string //"192.168.10.10/24"
	Gateway string
	DNS     []string
	NicCard string
}

func NetPlanFileStaticIp(spec NetworkSpec) string {
	content := quote.TemplateGeneric(quote.HereDoc(`
    network:
      ethernets:
        {{ .NicCard }}:
          addresses:
          - {{ .Address }}
          gateway4: {{ .Gateway }}
          nameservers:
            addresses:
            {{- range $i, $n := .DNS }}
            - {{ $n }}
            {{- end }}
      version: 2
  `), map[string]interface{}{
		"Address": spec.Address,
		"Gateway": spec.Gateway,
		"DNS":     spec.DNS,
		"NicCard": spec.NicCard,
	})
	// file := `netplan_config.yaml`
	// err = bastion.Put(content, file)

	log.Printf(content)
	return content
}

func SetHostname(hostName string) string {
	cmd := quote.CmdTemplate(`
    sudo sed -i -e '/^127.0.1.1/c\127.0.1.1 {{ .hostName }}' /etc/hosts
    sudo hostnamectl set-hostname {{ .hostName }}
  `, map[string]string{
		"hostName": hostName,
	})

	return cmd
}

func SudoNopass(pass string) string {
	cmd := quote.CmdTemplate(`
    echo {{ .pass }} | sudo -S sh -c "echo '%sudo ALL=(ALL:ALL) NOPASSWD: ALL' | sudo EDITOR='tee -a' visudo" 
  `, map[string]string{
		"pass": pass,
	})

	return cmd
}
