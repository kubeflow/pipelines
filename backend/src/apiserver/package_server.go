package main

import (
	"context"
	"ml/backend/api"
	"ml/backend/src/resource"
	"ml/backend/src/util"

	"github.com/golang/protobuf/ptypes/empty"
)

var packageModelFieldsBySortableAPIFields = map[string]string{
	"id":         "ID",
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
	return ToApiPackage(pkg), nil
}

func (s *PackageServer) ListPackages(ctx context.Context, request *api.ListPackagesRequest) (*api.ListPackagesResponse, error) {
	sortByModelField, ok := packageModelFieldsBySortableAPIFields[request.SortBy]
	if request.SortBy != "" && !ok {
		return nil, util.NewInvalidInputError("Received invalid sort by field %v.", request.SortBy)
	}
	packages, nextPageToken, err := s.resourceManager.ListPackages(request.PageToken, int(request.PageSize), sortByModelField)
	if err != nil {
		return nil, err
	}
	return &api.ListPackagesResponse{Packages: ToApiPackages(packages), NextPageToken: nextPageToken}, nil
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
