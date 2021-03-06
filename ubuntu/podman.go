package ubuntu

import "github.com/zhiminwen/quote"

//Ubuntu 20.04
func CmdToInstallPodman() string {
	cmd := quote.CmdTemplate(`
		. /etc/os-release
		sudo sh -c "echo 'deb https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/xUbuntu_${VERSION_ID}/ /' > /etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list"
		curl -L https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/xUbuntu_${VERSION_ID}/Release.key | sudo apt-key add -

		sudo apt update
		sudo apt install podman -y
`, map[string]string{})
	return cmd
}
