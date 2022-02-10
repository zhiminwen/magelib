package kvmrhel

import (
	"fmt"

	"github.com/zhiminwen/magetool/sshkit"
	"github.com/zhiminwen/quote"
)

type VMSpec struct {
	Name     string
	Cpu      int
	Mem      int //in GB
	Disk     int
	Bridge   string
	Password string

	Pool     string
	PoolPath string
	Network  string

	SourceImageFile string
	SshPublicKey    string

	IpCIDR      string //ip/24
	Gateway     string
	NameServers []string
}

func fill_default(vm VMSpec) VMSpec {
	vmNew := vm
	if vmNew.Password == "" {
		vmNew.Password = "password"
	}

	return vmNew
}

func cmd_createImage(vm VMSpec) string {
	cmd := quote.CmdTemplate(`
    qemu-img create -b {{ .sourceImage }} -f qcow2 {{ .poolPath }}/{{ .vmName }}.qcow2 {{ .diskSize }}G
  `, map[string]string{
		"sourceImage": vm.SourceImageFile,
		"poolPath":    vm.PoolPath,
		"vmName":      vm.Name,
	})

	return cmd
}

func cloudInit(vm VMSpec) string {
	content := quote.Template(quote.HereDoc(`
    #cloud-config
    hostname: {{ .vmName }}
    manage_etc_hosts: true
    users:
      - name: rhel
        sudo: ALL=(ALL) NOPASSWD:ALL
        groups: adm,sys
        home: /home/rhel
        shell: /bin/bash
        lock_passwd: false
        ssh-authorized-keys:
          - {{ .sshPublicKey }}
    ssh_pwauth: true
    disable_root: false
    chpasswd:
      list: |
        root: {{ .password }}
        rhel: {{ .passwor }}
      expire: False

    #runcmd:
    #		- [ sh, -c, 'sed -i s/BOOTPROTO=dhcp/BOOTPROTO=static/ /etc/sysconfig/network-scripts/ifcfg-eth0' ]
    #		- [ sh, -c, 'ifdown eth0 && sleep 1 && ifup eth0 && sleep 1 && ip a' ]
    
    
  `), map[string]string{
		"vmName":       vm.Name,
		"sshPublicKey": vm.SshPublicKey,
		"password":     vm.Password,
	})
	return content
}

func networkCfg(vm VMSpec) string {
	content := quote.Template(quote.HereDoc(`
    version: 2
    ethernets:
      eth0:
        dhcp4: false
        addresses: [ {{ .ipCidr }} ]
        gateway4: {{ .gw }}
        nameservers:
          addresses: [ {{ .dnsServers }} ]
  `), map[string]string{
		"ipCidr":     vm.IpCIDR,
		"gw":         vm.Gateway,
		"dnsServers": quote.Join(vm.NameServers, ","),
	})
	return content
}

func Provision_VM(sshClient *sshkit.SSHClient, vm VMSpec, workingDir string) error {
	vm = fill_default(vm)

	cmd := cmd_createImage(vm)
	err := sshClient.Execute(cmd)
	if err != nil {
		return err
	}

	content := cloudInit(vm)
	err = sshClient.Put(content, workingDir+"/"+vm.Name+".cloud-init.yaml")
	if err != nil {
		return err
	}

	content = networkCfg(vm)
	err = sshClient.Put(content, workingDir+"/"+vm.Name+".network.yaml")
	if err != nil {
		return err
	}

	cmd = quote.CmdTemplate(`
    cd {{ .dir }}
    cloud-localds -v --network-config={{ .vmName }}.network.yaml {{ .vmName }}-seed.qcow2 {{ .vmName }}.cloud-init.yaml
    
    virt-install --name={{ .vmName }} --ram={{ .mem }} --vcpus={{ .cpu }} --disk path={{ .path }}/{{ .vmName }}.qcow2,bus=virtio,cache=none --disk path={{ .vmName }}-seed.qcow2,device=cdrom --graphics=vnc --network network={{ .network }},model=virtio --boot hd
  `, map[string]string{
		"dir":      workingDir,
		"vmName":   vm.Name,
		"mem":      fmt.Sprintf("%d", vm.Mem*1024),
		"cpu":      fmt.Sprintf("%d", vm.Cpu),
		"diskSize": fmt.Sprintf("%dG", vm.Disk),
		"pool":     vm.Pool,
		"path":     vm.PoolPath,
		"network":  vm.Network,
	})
	err = sshClient.Execute(cmd)
	if err != nil {
		return err
	}
	return nil
}
