package tunnel

import (
	"log"

	"github.com/zhiminwen/magetool/shellkit"
	"github.com/zhiminwen/magetool/sshkit"
	"github.com/zhiminwen/quote"
)

type Tunnel struct {
	Remote     string //Ip or host
	TargetIp   string
	Port       string
	User       string
	Key        string
	RemoteUser string
}

func (tun *Tunnel) CreateTunnelSSH() *sshkit.SSHClient {
	sshClient, err := sshkit.NewSSHClient("localhost", tun.Port, tun.User, "", tun.Key)
	if err != nil {
		log.Fatalf("Failed to connect to LB:%v", err)
	}
	return sshClient
}

func (tun *Tunnel) CreateTunnel() {
	//assume the remote is using key authentication
	cmd := quote.CmdTemplate(`
		ssh -NL {{ .port }}:{{ .ip }}:22 {{ .user }}@{{ .remote }}& sleep 5
`, map[string]string{
		"port":   tun.Port,
		"ip":     tun.TargetIp,
		"remote": tun.Remote,
		"user":   tun.RemoteUser,
	})

	shellkit.ExecuteShell(cmd)
}

func (tun *Tunnel) CleanHostkey() {
	cmd := quote.CmdTemplate(`
		ssh-keygen -R "[localhost]:{{ .port }}" || echo no key found
	`, map[string]string{
		"port": tun.Port,
	})

	shellkit.ExecuteShell(cmd)
}

func (tun *Tunnel) Stop() {
	//this is for mac only
	cmd := quote.CmdTemplate(`
		ps -ef | grep 'ssh -NL' | grep -v grep| awk '{print $2}' | xargs kill
	`, map[string]string{})

	shellkit.ExecuteShell(cmd)
}

func New(remote, remoteUser, tunnelTargetIp, tunnelPort, tunnelUser, tunnelKey string) *Tunnel {
	tunnel := Tunnel{
		Remote:     remote,
		RemoteUser: remoteUser,
		TargetIp:   tunnelTargetIp,
		Port:       tunnelPort,
		User:       tunnelUser,
		Key:        tunnelKey,
	}

	return &tunnel
}
