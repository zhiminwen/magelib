package redhat

import (
	"log"

	"github.com/zhiminwen/magetool/sshkit"
	"github.com/zhiminwen/quote"
)

func SetStaticIP(sshServer *sshkit.SSHClient, iface, ip, prefix, gateway, dns string) {
	doc := quote.HereDoc(`
    DEVICE={{ .iface }}
    BOOTPROTO=none
    ONBOOT=yes
    PREFIX={{ .prefix }}
    IPADDR={{ .ip }}
    GATEWAY={{ .gateway }}
    DNS1={{ .dns }}
  `)

	content := quote.Template(doc, map[string]string{
		"iface":   iface,
		"prefix":  prefix,
		"ip":      ip,
		"gateway": gateway,
		"dns":     dns,
	})

	tmpFile := "/tmp/static.ip.config"
	err := sshServer.Put(content, tmpFile)
	if err != nil {
		log.Fatalf("Faile to upload iface file:%v", err)
	}
	cmds := quote.CmdTemplate(`
		// sudo cp -p {{ .ifaceFile }} /tmp/{{ .ifaceFile }}.$(date +%F)
 
		sudo cp {{ .tmpFile }} {{ .ifaceFile }}
    sudo rm {{ .tmpFile }}

    // sudo systemctl restart network
    // Cannot do restart network, otherwise the ssh session will be broken, and the go program hang.
  `, map[string]string{
		"tmpFile":   tmpFile,
		"ifaceFile": "/etc/sysconfig/network-scripts/ifcfg-" + iface,
	})

	sshServer.Execute(cmds)
}

func CreateVG(sshServer *sshkit.SSHClient, disk, vgName string) {
	cmd := quote.CmdTemplate(`
		sudo pvcreate {{ .disk }}
		sudo vgcreate {{ .vgName }} {{ .disk }}
	`, map[string]string{
		"disk":   disk,
		"vgName": vgName,
	})

	sshServer.Execute(cmd)
}
func CreateLV(sshServer *sshkit.SSHClient, size, lvName, vgName string) {
	// lvcreate -L 30G -n lv1 vg1
	cmd := quote.CmdTemplate(`
		sudo lvcreate -L {{ size }} -n {{ .lvName }} {{ .vgName }}
	`, map[string]string{
		"size":   size,
		"lvName": lvName,
		"vgName": vgName,
	})

	sshServer.Execute(cmd)
}

func CreateLVFull(sshServer *sshkit.SSHClient, lvName, vgName, fsType string) {
	cmd := quote.CmdTemplate(`
		sudo lvcreate -l100%VG -n {{ .lvName }} {{ .vgName }}
		sudo mkfs -t {{ .fsType}} /dev{{ .vgName }}/{{ .lvName }}
	`, map[string]string{
		"lvName": lvName,
		"vgName": vgName,
		"fsType": fsType,
	})

	sshServer.Execute(cmd)
}
