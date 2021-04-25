package virsh

import (
	"fmt"
	"log"

	"github.com/zhiminwen/magetool/sshkit"

	"github.com/zhiminwen/quote"
)

type VMSpec struct {
	Name     string
	Cpu      int
	Mem      int //in GB
	Disk     int
	Bridge   string
	Release  string
	Password string

	Pool     string
	PoolPath string
	Network  string

	IsoFile string

	ScriptPath string //for UVT run-once script
}

func fill_default(vm VMSpec) VMSpec {
	vmNew := vm
	if vmNew.Release == "" {
		vmNew.Release = "focal"
	}

	if vmNew.Password == "" {
		vmNew.Password = "password"
	}

	return vmNew
}

func By_uvt(sshClient *sshkit.SSHClient, vm VMSpec) error {
	vm = fill_default(vm)
	cmd := quote.CmdTemplate(`
    uvt-kvm create {{ .vmName }} release={{ .release }} --memory {{ .mem }} --cpu {{ .cpu }} --disk {{ .diskSize }} --bridge {{ .br }} --password password
`, map[string]string{
		"vmName":   vm.Name,
		"mem":      fmt.Sprintf("%d", vm.Mem*1024),
		"cpu":      fmt.Sprintf("%d", vm.Cpu),
		"diskSize": fmt.Sprintf("%d", vm.Disk),
		"br":       vm.Bridge,
		"release":  vm.Release, // "focal", //ubuntu 20.04
		"password": vm.Password,
	})

	if vm.ScriptPath != "" {
		cmd = fmt.Sprintf("%s --run-script-once %s", cmd, vm.ScriptPath)
	}

	log.Printf("cmd=%s", cmd)

	return sshClient.Execute(cmd)
}

func UVT_static_ip_script(nicName, staticIp, gateway string, dns []string) string {
	content := quote.TemplateGeneric(quote.HereDoc(`
    #! /bin/sh

    cat<<EOT>/etc/netplan/50-cloud-init.yaml
    network:
      ethernets:
          {{ .nicName }}:
            addresses:
            - {{ .staticIp }}
            gateway4: {{ .gateway }}
            nameservers:
              addresses:
              {{- range $i, $n := .dns }}
              - {{ $n }}
              {{- end }}
      version: 2
    EOT
    
    netplan apply
    
  `), map[string]interface{}{
		"nicName":  nicName,
		"staticIp": staticIp, //must have prefix such as /24
		"gateway":  gateway,
		"dns":      dns,
	})

	return content
}

func By_virt(sshClient *sshkit.SSHClient, vm VMSpec) error {
	var cdOption string
	if vm.IsoFile == "" {
		cdOption = "--disk device=cdrom"
	} else {
		cdOption = "--cdrom " + vm.IsoFile
	}
	cmd := quote.CmdTemplate(`
    virsh vol-create-as {{ .pool }} {{ .vmName }}.qcow2 {{.diskSize}}
    virt-install --name={{ .vmName }} --ram={{ .mem }} --vcpus={{ .cpu }} --disk path={{ .path }}/{{ .vmName }}.qcow2,bus=virtio,cache=none --pxe --noautoconsole --graphics=vnc --network network={{ .network }},model=virtio --boot hd,cdrom {{ .cdOption}}
  `, map[string]string{
		"vmName":   vm.Name,
		"mem":      fmt.Sprintf("%d", vm.Mem*1024),
		"cpu":      fmt.Sprintf("%d", vm.Cpu),
		"diskSize": fmt.Sprintf("%dG", vm.Disk),
		"pool":     vm.Pool,
		"path":     vm.PoolPath,
		"network":  vm.Network,
		"cdOption": cdOption,
	})

	return sshClient.Execute(cmd)
}

func Remove_uvt_vm(sshClient *sshkit.SSHClient, vmName string) error {
	cmd := quote.CmdTemplate(`
    uvt-kvm destroy {{ .vmName }} 
  `, map[string]string{
		"vmName": vmName,
	})

	return sshClient.Execute(cmd)
}

func Remove_kvm_vm(sshClient *sshkit.SSHClient, vmName string, pool, poolPath string) error {
	cmd := quote.CmdTemplate(`
      virsh destroy {{ .vmName }} || echo already down
      virsh undefine {{ .vmName }}

      rm -rf {{.path }}/{{ .vmName }}.qcow2
      virsh pool-refresh {{ .pool }}
    `, map[string]string{
		"vmName": vmName,
		"pool":   pool,
		"path":   poolPath,
	})

	return sshClient.Execute(cmd)
}

func Capture_mac(sshClient *sshkit.SSHClient, name string) (string, error) {
	cmd := quote.CmdTemplate(`
    // <mac address='52:54:00:0d:4a:a8'/>
    virsh dumpxml {{ .name }} | grep 'mac address' | cut -d\' -f 2
  `, map[string]string{
		"name": name,
	})
	mac, err := sshClient.Capture(cmd)
	if err != nil {
		return "", err
	}

	return mac, nil
}

func Find_dhcp_ip(sshClient *sshkit.SSHClient, mac string) (string, error) {
	cmd := quote.CmdTemplate(`
    //? (192.168.10.47) at 52:54:00:11:a3:96 [ether] on br-ocp
    arp -an | grep {{ .mac }} | cut -d \( -f 2 | cut -d \) -f 1 
`, map[string]string{
		"mac": mac,
	})
	ip, err := sshClient.Capture(cmd)
	if err != nil {
		return "", err
	}

	return ip, nil
}
