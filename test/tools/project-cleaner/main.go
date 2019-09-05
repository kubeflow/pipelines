// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/api/container/v1"
	"gopkg.in/yaml.v2"
)

// project_cleanup tool provides support to cleanup different resources insides a GCP project.
// Types of resources to be cleaned up in a single execution of the tool is specified as YAML file
// (run-time parameter flag) to the tool. Each GCP resource is associated with a handler function
// that performs cleanup. Adding a new resource for cleanup involves - adding a helper function to
// ProjectCleaner type and updating the switch condition in the CleanupProject method.

var (
	resourceSpec = flag.String("resource_spec", "", "resource spec yaml file")
)

// ResourceSpec is the top-level spec specifying GCP resources to be cleaned up in a project.
type ResourceSpec struct {
	// ProjectId - GCP Project Id
	ProjectId string `yaml:"project-id"`
	// Resources - Spec for each resource to be cleaned up by the tool.
	Resources []GCPResource `yaml:"resources"`
}

// GCPResource specifies an individual GCP resource to be cleaned up.
type GCPResource struct {
	// Resource - name of the resource is used as key to the handler function.
	Resource string `yaml:"resource"`
	// Zones - Multiple zones support.
	Zones []string `yaml:"zones"`
	// NamePrefixes - Only support for pre-fixed based filtering as go-client doesn't support regex
	// based filtering.
	NamePrefixes []string `yaml:"name-prefixes"`
	// TimeLapseInHours - Number of hours before the resource needs to be cleaned up.
	TimeLapseInHours int `yaml:"time-lapse-hours"`
}

// ProjectCleaner - top level type providing handlers for different GCP resources.
type ProjectCleaner struct {
	ProjectId string
	Resources []GCPResource
}

// GKEClusterHandler provides support to cleanup GCP resources within the project.
func (p *ProjectCleaner) GKEClusterHandler(resource GCPResource) {
	// See https://cloud.google.com/docs/authentication/.
	// Use GOOGLE_APPLICATION_CREDENTIALS environment variable to specify a service account key file
	// to authenticate to the API.
	ctx := context.Background()
	svc, err := container.NewService(ctx)
	if err != nil {
		log.Fatalf("Could not initialize gke client: %v", err)
	}

	elapsedTime := time.Now().Add(time.Hour * (-1) * time.Duration(resource.TimeLapseInHours))
	for _, zone := range resource.Zones {
		log.Printf("Listing gke clusters in zone: %v", zone)
		listResponse, err := svc.Projects.Zones.Clusters.List(p.ProjectId, zone).Do()
		if err != nil {
			log.Printf("Failed listing gke clusters in zone %v: %v", zone, err)
			continue
		}

		if listResponse == nil {
			log.Printf("Encountered nil listResponse gke clusters listing in zone: %v", zone)
			continue
		}

		for _, cluster := range listResponse.Clusters {
			createdTime, err := time.Parse(time.RFC3339, cluster.CreateTime)
			if err != nil {
				log.Printf("Unable to parse created time for cluster: %s. Ignoring cluster",
					cluster.Name)
				continue
			}

			if  p.checkForPrefix(cluster.Name, resource.NamePrefixes) && createdTime.Before(elapsedTime) {
				log.Printf("Found cluster: %s for deletion", cluster.Name)
				if op, err := svc.Projects.Zones.Clusters.Delete(p.ProjectId, zone, cluster.Name).Do();
						err != nil {
					log.Printf("Encountered error calling delete on cluster: %s Error: %v",
						cluster.Name, err)
				} else {
					log.Printf("Kicked off cluster delete : %v", op.TargetLink)
				}
			}
		}
	}
}

// checkForPrefix - helper function to check if testStr string has any of the prefix specified in
// prefixes
func (p *ProjectCleaner) checkForPrefix(testStr string, prefixes []string) bool {
	log.Printf("Performing prefix check for String: %s for Prefixes: %v", testStr, prefixes)
	for _, prefix := range prefixes {
		if strings.HasPrefix(testStr, prefix) {
			return true
		}
	}
	return false
}

// CleanupProject iterates over each resource specified in the spec and calls the corresponding
// cleanup handler method.
func (p *ProjectCleaner) CleanupProject() {
	for _, resource := range p.Resources {
		switch resource.Resource {
		case "gke-cluster":
			p.GKEClusterHandler(resource)
		default:
			log.Printf("Un-identified resource: %v found in spec. Ignoring", resource.Resource)
		}
	}
}

func main() {
	flag.Parse()
	log.SetOutput(os.Stdout)

	if *resourceSpec == "" {
		fmt.Fprintln(os.Stderr, "missing resource_spec parameter")
		flag.Usage()
		os.Exit(2)
	}

	if _, err := os.Stat(*resourceSpec); err != nil {
		log.Fatalf("missing resource spec file")
	}

	var r ResourceSpec
	yamlFile, err := ioutil.ReadFile(*resourceSpec)
	if err != nil {
		log.Fatalf("Error reading yaml file")
	}

	err = yaml.Unmarshal(yamlFile, &r)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	p := ProjectCleaner{
		ProjectId: r.ProjectId,
		Resources: r.Resources,
	}
	p.CleanupProject()
}
