package main

import (
	"encoding/json"
	"flag"
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

func statusValue(i string) int {
	switch i {
	case "SUCCESS":
		return 0
	case "WORKING":
		return 1
	case "QUEUED":
		return 2
	case "INTERNAL_ERROR":
		return 3
	case "CANCELLED":
		return 4
	case "TIMEOUT":
		return 5
	case "FAILURE":
		return 100
	}
	return 90
}

func (i *buildStepStatus) less(j *buildStepStatus) bool {
	n := statusValue(i.Status) - statusValue(j.Status)
	switch {
	case n > 0:
		return false
	case n < 0:
		return true
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

func parseBuildSteps(buildID string, projectID string) ([]buildStepStatus, error) {
	out, err := exec.Command("gcloud", "builds", "describe", buildID, "--format", "json", "--project", projectID).Output()
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
	var projectID string
	flag.StringVar(&projectID, "project", "", "project ID")
	flag.Parse()

	if projectID == "" {
		panic("ERROR: need project ID")
	}
	var bsss []buildStepStatus
	for _, buildID := range flag.Args() {
		s, err := parseBuildSteps(buildID, projectID)
		if err != nil {
			fmt.Printf("Error: %s: %v\n", buildID, err)
			continue
		}
		bsss = append(bsss, s...)
	}

	sort.SliceStable(bsss, func(i, j int) bool { return bsss[i].less(&bsss[j]) })
	failed := false
	for _, bss := range bsss {
		if strings.Compare(bss.Status, "SUCCESS") != 0 {
			failed = true
		}
		fmt.Printf("%-24s %s %s %s\n", bss.Name, bss.Tag, bss.ID, bss.Status)
	}

	if failed {
		os.Exit(1)
	}
}
