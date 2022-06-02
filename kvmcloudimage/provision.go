package kvmcloudimage

import (
	"fmt"
	"strings"

	"github.com/zhiminwen/magetool/sshkit"
	"github.com/zhiminwen/quote"
)

type VMSpec struct {
	Name   string
	Cpu    int
	Mem    int //in GB
	Disk   int
	Bridge string

	BaseUser      string //base admin user such as ubuntu, cloud-user
	Password      string
	SshPublicKeys []string

	SourceImageFile string //backing image
	PoolPath        string //where to store the image
	Network         string

	NicName     string
	IpCIDR      string //ip/24
	Gateway     string
	NameServers []string

	OsInfo string

	WorkingDir string //where to create the cloudinit iso image

	//for create vm with virt-install with iso
	IsoFile string //where to store the iso image
}

func fill_default(vm VMSpec) VMSpec {
	vmNew := vm
	if vmNew.Password == "" {
		vmNew.Password = "password"
	}
	if vmNew.WorkingDir == "" {
		vmNew.WorkingDir = "/tmp"
	}

	if vmNew.OsInfo == "" {
		vm.OsInfo = "linux2020"
	}

	return vmNew
}

func cloudInit(vm VMSpec) string {
	content := quote.TemplateGeneric(quote.HereDoc(`
    #cloud-config
    hostname: {{ .vmName }}
    fqdn: {{ .vmName }}
    manage_etc_hosts: false

    ssh_pwauth: true
    disable_root: false
    users:
      - default
      - name: {{ .user }}
        shell: /bin/bash
        sudo: ALL=(ALL) NOPASSWD:ALL
        lock_passwd: false
        ssh-authorized-keys:
          {{- range $i, $key := .pubKeys }}
          - {{ $key }}
          {{- end }}
    chpasswd:
      list: |
        root:{{ .password }}
        {{ .user }}:{{ .password }}
      expire: false
    runcmd:
      - [ sh, -c, echo {{ .ip }} {{ .hostname }} | tee -a /etc/hosts]
  `), map[string]interface{}{
		"vmName":   vm.Name,
		"user":     vm.BaseUser,
		"pubKeys":  vm.SshPublicKeys,
		"password": vm.Password,
		"ip":       strings.Split(vm.IpCIDR, "/")[0],
		"hostname": vm.Name,
	})
	return content
}

func networkCfg(vm VMSpec) string {
	content := quote.Template(quote.HereDoc(`
    version: 2
    ethernets:
      {{ .nicName }}:
        dhcp4: false
        addresses: [ {{ .ipCidr }} ]
        gateway4: {{ .gw }}
        nameservers:
          addresses: [ {{ .dnsServers }} ]
  `), map[string]string{
		"ipCidr":     vm.IpCIDR,
		"gw":         vm.Gateway,
		"dnsServers": strings.Join(vm.NameServers, ","),
		"nicName":    vm.NicName,
	})
	return content
}

func cmd_createImage(vm VMSpec) string {
	cmd := quote.CmdTemplate(`
    //need to define back file format also
    qemu-img create -b {{ .sourceImage }} -F qcow2 -f qcow2 {{ .poolPath }}/{{ .vmName }}.qcow2 {{ .diskSize }}G
  `, map[string]string{
		"sourceImage": vm.SourceImageFile,
		"poolPath":    vm.PoolPath,
		"vmName":      vm.Name,
		"diskSize":    fmt.Sprintf("%d", vm.Disk),
	})

	return cmd
}

func Provision_VM(sshClient *sshkit.SSHClient, vm VMSpec) error {
	vm = fill_default(vm)

	cmd := cmd_createImage(vm)
	err := sshClient.Execute(cmd)
	if err != nil {
		return err
	}

	content := cloudInit(vm)
	err = sshClient.Put(content, vm.WorkingDir+"/"+vm.Name+".cloud-init.yaml")
	if err != nil {
		return err
	}

	content = networkCfg(vm)
	err = sshClient.Put(content, vm.WorkingDir+"/"+vm.Name+".network.yaml")
	if err != nil {
		return err
	}

	cmd = quote.CmdTemplate(`
    cd {{ .dir }}
    cloud-localds -v --network-config={{ .vmName }}.network.yaml {{ .vmName }}-seed.qcow2 {{ .vmName }}.cloud-init.yaml
    
    virt-install --name={{ .vmName }} --ram={{ .mem }} --vcpus={{ .cpu }} --disk path={{ .path }}/{{ .vmName }}.qcow2,bus=virtio,cache=none --disk path={{ .vmName }}-seed.qcow2,device=cdrom --noautoconsole --graphics=vnc --network network={{ .network }},model=virtio --boot hd --osinfo {{ .osInfo }}
  `, map[string]string{
		"dir":      vm.WorkingDir,
		"vmName":   vm.Name,
		"mem":      fmt.Sprintf("%d", vm.Mem*1024),
		"cpu":      fmt.Sprintf("%d", vm.Cpu),
		"diskSize": fmt.Sprintf("%dG", vm.Disk),
		"path":     vm.PoolPath,
		"network":  vm.Network,
		"osInfo":   vm.OsInfo,
	})
	err = sshClient.Execute(cmd)
	if err != nil {
		return err
	}
	return nil

}

//Eject only when the cloudinit boot is finished
func Eject_CloudInit_CD(sshClient *sshkit.SSHClient, vm VMSpec) error {
	cmd := quote.CmdTemplate(`
    cd {{ .dir }}
    virsh change-media {{ .vmName }} --path $(readlink -f {{ .vmName }}-seed.qcow2) --eject --force

		//remove seed disk
		rm -rf {{ .vmName }}-seed.qcow2 
  `, map[string]string{
		"dir":    vm.WorkingDir,
		"vmName": vm.Name,
	})
	err := sshClient.Execute(cmd)
	if err != nil {
		return err
	}
	return nil
}

func Delete_VM(sshClient *sshkit.SSHClient, vm VMSpec) error {
	cmd := quote.CmdTemplate(`
    virsh destroy {{ .vmName }} || echo ignore not running
    virsh undefine {{ .vmName }} 
    rm -rf {{ .poolPath }}/{{ .vmName }}.qcow2
  `, map[string]string{
		"vmName":   vm.Name,
		"poolPath": vm.PoolPath,
	})
	err := sshClient.Execute(cmd)
	if err != nil {
		return err
	}
	return nil
}

//create VM with virt-install
func Create_VM_with_Virt_Install(sshClient *sshkit.SSHClient, vm VMSpec) error {
	var cdOption, osInfo string

	if vm.IsoFile == "" {
		cdOption = "--disk device=cdrom"
	} else {
		cdOption = "--cdrom " + vm.IsoFile
	}

	if vm.OsInfo == "" {
		osInfo = ""
	} else {
		osInfo = "--osinfo " + vm.OsInfo
	}
	cmd := quote.CmdTemplate(`
		// virsh vol-create-as {{ .pool }} {{ .vmName }}.qcow2 {{.diskSize}}
		qemu-img create -f qcow2 {{ .path }}/{{ .vmName }}.qcow2 {{ .diskSize }}
		virt-install --name={{ .vmName }} --ram={{ .mem }} --vcpus={{ .cpu }} --disk path={{ .path }}/{{ .vmName }}.qcow2,bus=virtio,cache=none --noautoconsole --graphics=vnc --network network={{ .network }},model=virtio --boot hd,cdrom {{ .cdOption}} {{ .osInfo }}
`, map[string]string{
		"vmName":   vm.Name,
		"mem":      fmt.Sprintf("%d", vm.Mem*1024),
		"cpu":      fmt.Sprintf("%d", vm.Cpu),
		"diskSize": fmt.Sprintf("%dG", vm.Disk),
		"path":     vm.PoolPath,
		"network":  vm.Network,
		"cdOption": cdOption,
		"osInfo":   osInfo,
	})

	return sshClient.Execute(cmd)
}

//osinfo is now a must for Ubuntu 22.04
