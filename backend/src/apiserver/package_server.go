package main

import (
	"context"
	"ml/backend/api"
	"ml/backend/src/resource"

	"github.com/golang/protobuf/ptypes/empty"
)

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
	packages, err := s.resourceManager.ListPackages()
	if err != nil {
		return nil, err
	}
	return &api.ListPackagesResponse{Packages: ToApiPackages(packages)}, nil
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
