This will replace the uvt-kvm way of creating KVM vm

## Sample usage
```go
func get_redhat_vm_spec() kvmcloudimage.VMSpec {
	pubKey, err := os.ReadFile(".ssh/id_rsa.pub")
	if err != nil {
		log.Fatalf("failed to read pub key:%v", err)
	}

	return kvmcloudimage.VMSpec{
		Name: "cl-redhat",
		Cpu:  4,
		Mem:  8,
		Disk: 100,

		BaseUser: "rhel",
		SshPublicKeys: []string{
			string(pubKey),
		},

		PoolPath: "/data1/kvm-images",
		Network:  "ocp",

		SourceImageFile: "/data1/kvm-images/rhel-8.5-update-2-x86_64-kvm.qcow2",

		NicName:     "eth0",
		IpCIDR:      "192.168.10.123/24",
		Gateway:     "192.168.10.1",
		NameServers: quote.Word(`10.0.80.11 10.0.80.12`),

		WorkingDir: "/tmp",
	}
}

func (Redhat) T01_provision() {
	kvmcloudimage.Provision_VM(master, get_redhat_vm_spec())
}

func (Redhat) T02_eject_cd() {
	kvmcloudimage.Eject_CloudInit_CD(master, get_redhat_vm_spec())
}

func (Redhat) T10_delete() {
	kvmcloudimage.Delete_VM(master, get_redhat_vm_spec())
}
```