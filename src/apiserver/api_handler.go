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
	"io/ioutil"
	"ml/src/apiserver/api"
	"ml/src/model"
	"ml/src/resource"
	"ml/src/util"

	"fmt"

	"github.com/golang/glog"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

const (
	apiRouterPrefix = "/apis/v1alpha1"

	healthz = "/healthz"

	listPackages  = "/packages"
	getPackage    = "/packages/{packageID:long min(1)}"
	uploadPackage = "/packages/upload"
	getTemplate   = "/packages/{packageID:long min(1)}/templates"

	listPipelines   = "/pipelines"
	getPipeline     = "/pipelines/{pipelineID:long min(1)}"
	createPipeline  = "/pipelines"
	enablePipeline  = "/pipelines/{pipelineID:long min(1)}/enable"
	disablePipeline = "/pipelines/{pipelineID:long min(1)}/disable"

	listJobs = "/pipelines/{pipelineID:long min(1)}/jobs"
	getJob   = "/pipelines/{pipelineID:long min(1)}/jobs/{jobName:string}"
)

type apiResponse interface{}

type APIHandler struct {
	resourceManager *resource.ResourceManager
}

func (a APIHandler) ListPackages(ctx iris.Context) (apiResponse, error) {
	glog.Infof("List packages called")

	packages, err := a.resourceManager.ListPackages()
	if err != nil {
		return nil, err
	}

	return api.ToApiPackages(packages), nil
}

func (a APIHandler) GetPackage(ctx iris.Context) (apiResponse, error) {
	glog.Infof("Get package called")

	packageID, err := getInt64ID(ctx, "packageID")
	if err != nil {
		return nil, err
	}

	pkg, err := a.resourceManager.GetPackage(uint(packageID))
	if err != nil {
		return nil, err
	}

	return api.ToApiPackage(pkg), nil
}

// Stream the file to API server. This is OK for now since we only support YAML file which is small.
// TODO(yangpa): In near future, use Minio Presigned Put instead.
// For more info check https://docs.minio.io/docs/golang-client-api-reference#PresignedPutObject
func (a APIHandler) UploadPackage(ctx iris.Context) (apiResponse, error) {
	glog.Infof("Upload package called")

	// Get the file from the request.
	file, info, err := ctx.FormFile("uploadfile")
	if err != nil {
		return nil, util.NewInvalidInputError(err, "Failed to read package.", err.Error())
	}

	defer file.Close()

	// Read file to byte array
	pkgFile, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, util.NewBadRequestError(err, "Error while uploading: <%s>.", err.Error())
	}

	// Store the package file
	err = a.resourceManager.CreatePackageFile(pkgFile, info.Filename)
	if err != nil {
		return nil, err
	}

	// Extract the parameter from the package
	params, err := util.GetParameters(pkgFile)
	if err != nil {
		return nil, err
	}
	pkg := &model.Package{Name: info.Filename, Parameters: params}

	newPkg, err := a.resourceManager.CreatePackage(pkg)
	if err != nil {
		return nil, err
	}
	return api.ToApiPackage(newPkg), nil
}

func (a APIHandler) GetTemplate(ctx iris.Context) (apiResponse, error) {
	glog.Infof("Get template called")

	packageID, err := getInt64ID(ctx, "packageID")
	if err != nil {
		return nil, err
	}

	pkg, err := a.resourceManager.GetPackage(uint(packageID))
	if err != nil {
		return nil, err
	}

	template, err := a.resourceManager.GetTemplate(pkg.Name)
	if err != nil {
		return nil, err
	}
	// Write directly to context.
	ctx.Write(template)
	return nil, nil
}

func (a APIHandler) ListPipelines(ctx iris.Context) (apiResponse, error) {
	glog.Infof("List pipelines called")

	pipelines, err := a.resourceManager.ListPipelines()
	if err != nil {
		return nil, err
	}

	return api.ToApiPipelines(pipelines), nil
}

func (a APIHandler) GetPipeline(ctx iris.Context) (apiResponse, error) {
	glog.Infof("Get pipeline called")

	pipelineID, err := getInt64ID(ctx, "pipelineID")
	if err != nil {
		return nil, err
	}

	pipeline, err := a.resourceManager.GetPipeline(uint(pipelineID))
	if err != nil {
		return nil, err
	}

	return api.ToApiPipeline(pipeline), nil
}

func (a APIHandler) CreatePipeline(ctx iris.Context) (apiResponse, error) {
	glog.Infof("Create pipeline called")

	pipeline := &api.Pipeline{}
	if err := ctx.ReadJSON(pipeline); err != nil {
		return nil, util.NewInvalidInputError(err, "The pipeline has invalid format.", err.Error())
	}

	newPipeline, err := a.resourceManager.CreatePipeline(api.ToModelPipeline(pipeline))
	if err != nil {
		return nil, err
	}

	return api.ToApiPipeline(newPipeline), nil
}

func (a APIHandler) EnablePipeline(ctx iris.Context) (apiResponse, error) {
	return a.enablePipeline(ctx, true)
}

func (a APIHandler) DisablePipeline(ctx iris.Context) (apiResponse, error) {
	return a.enablePipeline(ctx, false)
}

func (a APIHandler) enablePipeline(ctx iris.Context, enabled bool) (apiResponse, error) {
	glog.Infof("Enable pipeline")

	pipelineID, err := getInt64ID(ctx, "pipelineID")
	if err != nil {
		return nil, err
	}

	err = a.resourceManager.EnablePipeline(uint(pipelineID), enabled)
	if err != nil {
		return nil, err
	}
	// No response to return
	return nil, nil
}

func (a APIHandler) ListJobs(ctx iris.Context) (apiResponse, error) {
	glog.Infof("List jobs called")

	pipelineID, err := getInt64ID(ctx, "pipelineID")
	if err != nil {
		return nil, err
	}

	jobs, err := a.resourceManager.ListJobs(uint(pipelineID))
	if err != nil {
		return nil, err
	}

	return api.ToApiJobs(jobs), nil
}

func (a APIHandler) GetJob(ctx iris.Context) (apiResponse, error) {
	glog.Infof("Get job called")

	pipelineID, err := getInt64ID(ctx, "pipelineID")
	if err != nil {
		return nil, err
	}

	jobName := ctx.Params().Get("jobName")

	jobDetails, err := a.resourceManager.GetJob(uint(pipelineID), jobName)
	if err != nil {
		return nil, err
	}

	return api.ToApiJobDetail(jobDetails), nil
}

func getInt64ID(ctx iris.Context, idName string) (int64, error) {
	id, err := ctx.Params().GetInt64(idName)
	if err != nil {
		return id,
			util.NewInvalidInputError(err, fmt.Sprintf("The %s is invalid.", idName), err.Error())
	}
	return id, nil
}

func newApp(clientManager ClientManager) *iris.Application {

	apiHandler := &APIHandler{
		resourceManager: resource.NewResourceManager(&clientManager),
	}

	app := iris.New()
	// registers a custom handler for 404 not found http (error) status code,
	// fires when route not found or manually by ctx.StatusCode(iris.StatusNotFound).
	app.OnErrorCode(iris.StatusNotFound, notFoundHandler)

	apiRouter := app.Party(apiRouterPrefix)

	// Packages
	apiRouter.Get(listPackages, newHandler(apiHandler.ListPackages))
	apiRouter.Get(getPackage, newHandler(apiHandler.GetPackage))
	apiRouter.Post(uploadPackage, newHandler(apiHandler.UploadPackage))
	apiRouter.Get(getTemplate, newHandler(apiHandler.GetTemplate))

	// Pipelines
	apiRouter.Get(listPipelines, newHandler(apiHandler.ListPipelines))
	apiRouter.Get(getPipeline, newHandler(apiHandler.GetPipeline))
	apiRouter.Post(createPipeline, newHandler(apiHandler.CreatePipeline))
	apiRouter.Options(createPipeline, func(iris.Context) {})
	apiRouter.Post(enablePipeline, newHandler(apiHandler.EnablePipeline))
	apiRouter.Post(disablePipeline, newHandler(apiHandler.DisablePipeline))

	// Jobs
	apiRouter.Get(listJobs, newHandler(apiHandler.ListJobs))
	apiRouter.Get(getJob, newHandler(apiHandler.GetJob))

	// Monitoring
	apiRouter.Get(healthz, func(iris.Context) {})
	return app
}

type innerHandler func(ctx iris.Context) (response apiResponse, err error)

func newHandler(handler innerHandler) context.Handler {
	return func(ctx context.Context) {
		response, err := handler(ctx)
		if err != nil {
			util.PopulateContextAndLogError(ctx, err)
			return
		}
		if response != nil {
			ctx.JSON(response)
		}
	}
}

func notFoundHandler(ctx iris.Context) {
	ctx.HTML("Nothing is here.")
}
