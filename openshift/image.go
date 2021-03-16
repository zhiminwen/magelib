package openshift

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zhiminwen/quote"
)

type ImageData struct {
	Reg    string
	Repo   string
	Digest string
	Tag    string
	Type   string
}

func CreateImgData(lists []string) []ImageData {
	// List Sample:
	// docker.io/ceph/ceph:v15.2.4
	// 		Reg: docker.io
	// 		Repo: /ceph/ceph
	// 		tag: v15.2.4
	result := []ImageData{}
	for _, line := range lists {
		data := strings.SplitN(line, "/", 2)
		reg := data[0]
		repoTags := strings.Split(data[1], ":")

		var repo, tag string
		if len(repoTags) > 1 {
			repo, tag = repoTags[0], repoTags[1]
		} else {
			repo = repoTags[0]
			tag = "" //latest
		}

		result = append(result, ImageData{
			Reg:  reg,
			Repo: repo,
			Tag:  tag,
		})
	}

	return result
}

type Repo struct {
	Registry string
	Repo     string
}

func getRepo(imgData []ImageData) map[string]Repo {
	repoList := map[string]Repo{}
	for _, img := range imgData {
		key := fmt.Sprintf("%s/%s", img.Reg, filepath.Dir(img.Repo))
		repoList[key] = Repo{
			Registry: img.Reg,
			Repo:     filepath.Dir(img.Repo),
		}
	}
	return repoList
}

//create ocp ImageContentSourcePolicy for airgap images
func GenOCPImagePolicyYaml(imgData []ImageData, name string, localRegUrl string) string {
	repoList := getRepo(imgData)
	content := quote.TemplateGeneric(quote.HereDoc(`
    apiVersion: operator.openshift.io/v1alpha1
    kind: ImageContentSourcePolicy
    metadata:
      name: {{ .name }}
    spec:
      repositoryDigestMirrors:
      {{ range $key, $repo := .mirrors -}}
      - mirrors:
        - {{ $.localRegUrl }}/{{ $repo.Repo }}
        source: {{ $key }}
      {{ end }}
  `), map[string]interface{}{
		"name":        name,
		"mirrors":     repoList,
		"localRegUrl": localRegUrl,
	})

	return content
}
