package coreos

import "github.com/zhiminwen/quote"

func CmdToRemoveImage(node string, imgRegex string) string {
	//must run on the Bastion node where it can access the worker nodes
	cmd := quote.CmdTemplate(`
		export IMAGEID=$(ssh {{ .sshOptions }} core@{{ .node }} sudo crictl images | grep {{ .imgRegex }} | awk '{print $3}')
		if [ -z "${IMAGEID}" ]; then echo no such image; else ssh {{ .sshOptions }} core@{{ .node }} sudo crictl rmi $IMAGEID; fi
`, map[string]string{
		"sshOptions": `-o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no"`,
		"node":       node,
		"imgRegex":   imgRegex,
	})

	return cmd
}
