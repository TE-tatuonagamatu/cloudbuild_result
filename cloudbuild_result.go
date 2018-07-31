package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"google.golang.org/api/cloudbuild/v1"
)

type buildStepStatus struct {
	Name   string
	Tag    string
	ID     string
	Status string
}

func (i *buildStepStatus) less(j *buildStepStatus) bool {
	switch strings.Compare(i.Status, j.Status) {
	case 1:
		return true
	case -1:
		return false
	}
	switch strings.Compare(i.Name, j.Name) {
	case 1:
		return false
	case -1:
		return true
	}
	switch strings.Compare(i.ID, j.ID) {
	case 1:
		return false
	case -1:
		return true
	}
	return strings.Compare(i.Tag, j.Tag) > 0
}

func getTagName(env []string) string {
	tag := ""
	for _, e := range env {
		es := strings.Split(e, "=")
		if len(es) < 2 {
			continue
		}
		if strings.Compare(es[0], "REMOTE_TAG_NAME") == 0 {
			tag = es[1]
			break
		}
	}
	return tag
}

func parseBuildSteps(buildID string) ([]buildStepStatus, error) {
	out, err := exec.Command("gcloud", "builds", "describe", buildID, "--format", "json").Output()
	if err != nil {
		return nil, err
	}

	var build cloudbuild.Build
	err = json.Unmarshal(out, &build)
	if err != nil {
		return nil, err
	}

	var infos []buildStepStatus
	for _, s := range build.Steps {
		switch {
		case strings.Compare(s.Id, "prolog") == 0:
		case strings.Compare(s.Id, "pull-builder-image") == 0:
		default:
			infos = append(infos, buildStepStatus{Name: s.Id, Status: s.Status, Tag: getTagName(s.Env), ID: build.Id})
		}
	}
	return infos, nil
}

func main() {
	var bsss []buildStepStatus
	for i, buildID := range os.Args {
		if i == 0 {
			continue
		}
		s, err := parseBuildSteps(buildID)
		if err != nil {
			fmt.Printf("Error: %s: %v\n", buildID, err)
			continue
		}
		bsss = append(bsss, s...)
	}

	sort.SliceStable(bsss, func(i, j int) bool { return bsss[i].less(&bsss[j]) })
	for _, bss := range bsss {
		fmt.Printf("%-22s\t%s\t%s\t%s\n", bss.Name, bss.Tag, bss.ID, bss.Status)
	}
}
