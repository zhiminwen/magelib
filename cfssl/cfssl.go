package cfssl

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/zhiminwen/magetool/shellkit"
	"github.com/zhiminwen/quote"
)

type CFSSLTool struct {
	WorkingDir string
}

func NewCFSSLTool(workDir string) (*CFSSLTool, error) {
	//43800h = 5 year
	doc := quote.HereDoc(`
    {
      "signing": {
          "default": {
              "expiry": "43800h"
          },
          "profiles": {
              "server": {
                  "expiry": "43800h",
                  "usages": [
                      "signing",
                      "key encipherment",
                      "server auth",
                      "client auth"
                  ]
              },
              "client": {
                  "expiry": "43800h",
                  "usages": [
                      "signing",
                      "key encipherment",
                      "client auth"
                  ]
              }
          }
      }
    }
  `)
	err := os.MkdirAll(workDir, 0755)
	if err != nil {
		log.Printf("Failed to create wokring dir:%v", err)
		return nil, err
	}

	err = ioutil.WriteFile(workDir+"/ca-config.json", []byte(doc), 0644)
	if err != nil {
		log.Printf("Failed to create ca config file:%v", err)
		return nil, err
	}

	return &CFSSLTool{
		WorkingDir: workDir,
	}, nil
}

func (cfssltool *CFSSLTool) CreateSelfSignedCA(cn string, listOfHosts []string) {
	doc := quote.HereDoc(`
      {
         "CN": "{{ .cn }}",
         "hosts": [ {{ .listOfHosts }} ],
         "key": {
            "algo": "rsa",
            "size": {{ .keySize }}
         },
         "names": [
            {
                "C": "SG",
                "ST": "SG",
                "L": "Singapore"
            }
        ]
      }
    `)

	list := []string{}
	for _, host := range listOfHosts {
		list = append(list, fmt.Sprintf(`"%s"`, host))
	}
	content := quote.Template(doc, map[string]string{
		"cn":          cn,
		"keySize":     "4096",
		"listOfHosts": strings.Join(list, ","),
	})

	ioutil.WriteFile(cfssltool.WorkingDir+"/myca.json", []byte(content), 0644)

	cmd := quote.CmdTemplate(`
		cd {{ .dir }}
		cfssl gencert -initca myca.json | cfssljson -bare myca
	`, map[string]string{
		"dir": cfssltool.WorkingDir,
	})
	shellkit.ExecuteShell(cmd)
}

func (cfssltool *CFSSLTool) CreateClientCert(cn string, listOfHosts []string, certName string) {
	doc := quote.HereDoc(`
      {
         "CN": "{{ .cn }}",
         "hosts": [ {{ .listOfHosts }} ],
         "key": {
            "algo": "rsa",
            "size": {{ .keySize }}
         }
      }
    `)

	list := []string{}
	for _, host := range listOfHosts {
		list = append(list, fmt.Sprintf(`"%s"`, host))
	}
	content := quote.Template(doc, map[string]string{
		"cn":          cn,
		"keySize":     "4096",
		"listOfHosts": strings.Join(list, ","),
	})

	ioutil.WriteFile(cfssltool.WorkingDir+"/clientRequest.json", []byte(content), 0644)
	cmd := quote.CmdTemplate(`
    // set PATH=%PATH%;c:\Tools\cfssl
    cd {{ .workDir }}
    cfssl gencert -ca=myca.pem -ca-key=myca-key.pem -config=ca-config.json -profile=client -hostname={{ .listOfHosts }} clientRequest.json | cfssljson -bare {{ .certName }}
   
  `, map[string]string{
		"workDir":     cfssltool.WorkingDir,
		"certName":    certName,
		"listOfHosts": strings.Join(listOfHosts, ","),
	})

	shellkit.ExecuteShell(cmd)
}

func (cfssltool *CFSSLTool) CreateServerCert(cn string, listOfHosts []string, certName string) {
	doc := quote.HereDoc(`
      {
         "CN": "{{ .cn }}",
         "hosts": [ {{ .listOfHosts }} ],
         "key": {
            "algo": "rsa",
            "size": {{ .keySize }}
         }
      }
    `)

	list := []string{}
	for _, host := range listOfHosts {
		list = append(list, fmt.Sprintf(`"%s"`, host))
	}
	content := quote.Template(doc, map[string]string{
		"cn":          cn,
		"keySize":     "4096",
		"listOfHosts": strings.Join(list, ","),
	})

	//create server key
	err := ioutil.WriteFile(cfssltool.WorkingDir+"/serverRequest.json", []byte(content), 0644)
	if err != nil {
		log.Fatalf("Failed to save server cert request:%v", err)
	}

	cmd := quote.CmdTemplate(`
    // set PATH=%PATH%;c:\Tools\cfssl
    cd {{ .workDir }}
    cfssl gencert -ca=myca.pem -ca-key=myca-key.pem -config=ca-config.json -profile=server -hostname={{ .listOfHosts }} serverRequest.json | cfssljson -bare {{ .certName }} 
  `, map[string]string{
		"workDir":     cfssltool.WorkingDir,
		"certName":    certName,
		"listOfHosts": strings.Join(listOfHosts, ","), //without "
	})

	shellkit.ExecuteShell(cmd)
}
