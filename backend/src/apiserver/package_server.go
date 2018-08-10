// Copyright 2018 Google LLC
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
)

var packageModelFieldsBySortableAPIFields = map[string]string{
	// Sort by CreatedAtInSec by default
	"":           "CreatedAtInSec",
	"id":         "UUID",
	"name":       "Name",
	"created_at": "CreatedAtInSec",
}

type PackageServer struct {
	resourceManager *resource.ResourceManager
}

func (s *PackageServer) GetPackage(ctx context.Context, request *api.GetPackageRequest) (*api.Package, error) {
	pkg, err := s.resourceManager.GetPackage(request.Id)
	if err != nil {
		return nil, err
	}
	return ToApiPackage(pkg)
}

func (s *PackageServer) ListPackages(ctx context.Context, request *api.ListPackagesRequest) (*api.ListPackagesResponse, error) {
	sortByModelField, isDesc, err := parseSortByQueryString(request.SortBy, packageModelFieldsBySortableAPIFields)
	if err != nil {
		return nil, err
	}
	packages, nextPageToken, err := s.resourceManager.ListPackages(request.PageToken, int(request.PageSize), sortByModelField, isDesc)
	if err != nil {
		return nil, err
	}
	apiPackages, err := ToApiPackages(packages)
	if err != nil {
		return nil, err
	}
	return &api.ListPackagesResponse{Packages: apiPackages, NextPageToken: nextPageToken}, nil
}

func (s *PackageServer) DeletePackage(ctx context.Context, request *api.DeletePackageRequest) (*empty.Empty, error) {
	err := s.resourceManager.DeletePackage(request.Id)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func (s *PackageServer) GetTemplate(ctx context.Context, request *api.GetTemplateRequest) (*api.GetTemplateResponse, error) {
	template, err := s.resourceManager.GetPackageTemplate(request.Id)
	if err != nil {
		return nil, err
	}

	return &api.GetTemplateResponse{Template: string(template)}, nil
}
