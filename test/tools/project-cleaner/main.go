// Copyright 2019 The Kubeflow Authors
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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"google.golang.org/api/container/v1"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
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

			if p.checkForPrefix(cluster.Name, resource.NamePrefixes) && createdTime.Before(elapsedTime) {
				log.Printf("Found cluster: %s for deletion", cluster.Name)
				if op, err := svc.Projects.Zones.Clusters.Delete(p.ProjectId, zone, cluster.Name).Do(); err != nil {
					log.Printf("Encountered error calling delete on cluster: %s Error: %v",
						cluster.Name, err)
				} else {
					log.Printf("Kicked off cluster delete : %v", op.TargetLink)
				}
			}
		}
	}
}

func (p *ProjectCleaner) PersistentDiskHandler(resource GCPResource) {
	// See https://cloud.google.com/docs/authentication/.
	// Use GOOGLE_APPLICATION_CREDENTIALS environment variable to specify a service account key file
	// to authenticate to the API.
	ctx := context.Background()
	c, err := compute.NewDisksRESTClient(ctx)
	if err != nil {
		log.Fatalf("Could not initialize perisistent disk client: %v", err)
	}
	defer c.Close()

	for _, zone := range resource.Zones {
		filter := `(lastDetachTimestamp != "")`
		// maxResults := uint32(5)
		// order := "creationTimestamp desc"
		req := &computepb.ListDisksRequest{
			Project: p.ProjectId,
			Zone:    zone,
			Filter:  &filter,
			// OrderBy:    &order,
			// MaxResults: &maxResults,
		}
		disk_list, listerr := c.List(ctx, req)
		if listerr != nil {
			log.Fatalf("Could not get perisistent disk list: %v, zone: %s", listerr, zone)
		}
		diskmarshal, _ := json.Marshal(disk_list)
		log.Printf("disk_list:%s", string(diskmarshal))
		log.Printf("disk_list size:%d", len(disk_list.GetItems()))

		for _, disk := range disk_list.GetItems() {
			// If the disk is not detached yet, do not delete the persistent disk.
			if len(disk.GetLastDetachTimestamp()) == 0 {
				log.Printf("disk: %s doesn't have last detach time, do not delete.", disk.GetName())
				continue
			}
			log.Printf("disk: %s last detach time: %s", disk.GetName(), disk.GetLastDetachTimestamp())

			// If the disk has users (attached instances) in form: projects/project/zones/zone/instances/instance, do not delete the persistent disk.
			if len(disk.GetUsers()) != 0 {
				log.Printf("disk: %s has users: %s, do not delete.", disk.GetName(), disk.GetUsers())
				continue
			}

			// Do not delete the persistent disk if this is less than specified hours old.
			creationTimestamp := disk.GetCreationTimestamp()
			createdTime, _ := time.Parse(time.RFC3339, creationTimestamp)
			duration := time.Since(createdTime)
			elapsedTime := time.Now().Add(time.Hour * (-1) * time.Duration(resource.TimeLapseInHours))
			if createdTime.After(elapsedTime) {
				log.Printf("disk: %s has been: %v old, do not delete.", disk.GetName(), duration)
				continue
			}
			log.Printf("disk: %s has been: %v old, should be deleted.", disk.GetName(), duration)
			log.Printf("Deleting disk: %s", disk.GetName())

			delete_req := &computepb.DeleteDiskRequest{
				Disk:    disk.GetName(),
				Project: p.ProjectId,
				Zone:    zone,
			}
			_, delete_err := c.Delete(ctx, delete_req)
			if delete_err != nil {
				log.Fatalf("Could not delete perisistent disk: %v", delete_err)
			}
			log.Printf("Deleted disk: %s", disk.GetName())
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
		case "disk":
			p.PersistentDiskHandler(resource)
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
