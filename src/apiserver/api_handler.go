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
	"fmt"
	"io/ioutil"
	"ml/src/message"
	"ml/src/resource"
	"ml/src/util"

	"github.com/golang/glog"
	"github.com/kataras/iris"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
)

const (
	apiRouterPrefix = "/apis/v1alpha1"

	listPackages  = "/packages"
	getPackage    = "/packages/{id:long min(1)}"
	uploadPackage = "/packages/upload"
	getTemplate   = "/packages/{id:long min(1)}/templates"

	listPipelines   = "/pipelines"
	getPipeline     = "/pipelines/{pipelineId:long min(1)}"
	createPipeline  = "/pipelines"
	enablePipeline  = "/pipelines/{pipelineId:long min(1)}/enable"
	disablePipeline = "/pipelines/{pipelineId:long min(1)}/disable"

	listJobs = "/pipelines/{pipelineId:long min(1)}/jobs"
	getJob   = "/pipelines/{pipelineId:long min(1)}/jobs/{jobName:string}"
)

type APIHandler struct {
	resourceManager *resource.ResourceManager
}

func (a APIHandler) ListPackages(ctx iris.Context) {
	glog.Infof("List packages called")

	packages, err := a.resourceManager.GetPackageStore().ListPackages()
	if err != nil {
		util.HandleError("ListPackages", ctx, err)
		return
	}

	ctx.JSON(packages)
}

func (a APIHandler) GetPackage(ctx iris.Context) {
	glog.Infof("Get package called")

	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		util.HandleError("GetPackage_GetParam", ctx, util.NewInvalidInputError("The package ID is invalid.", err.Error()))
		return
	}
	pkg, err := a.resourceManager.GetPackageStore().GetPackage(uint(id))

	if err != nil {
		util.HandleError("GetPackage", ctx, err)
		return
	}

	ctx.JSON(pkg)
}

// Stream the file to API server. This is OK for now since we only support YAML file which is small.
// TODO(yangpa): In near future, use Minio Presigned Put instead.
// For more info check https://docs.minio.io/docs/golang-client-api-reference#PresignedPutObject
func (a APIHandler) UploadPackage(ctx iris.Context) {
	glog.Infof("Upload package called")

	// Get the file from the request.
	file, info, err := ctx.FormFile("uploadfile")

	if err != nil {
		util.HandleError("UploadPackage_GetFormFile ", ctx, util.NewInvalidInputError("Failed to read package.", err.Error()))
		return
	}

	defer file.Close()

	// Read file to byte array
	pkgFile, err := ioutil.ReadAll(file)
	if err != nil {
		util.HandleError("UploadPackage_ReadFile", ctx, util.NewInternalError("Failed to read package.", err.Error()))
		return
	}

	// Store the package file
	err = a.resourceManager.GetPackageManager().CreatePackageFile(pkgFile, info.Filename)
	if err != nil {
		util.HandleError("UploadPackage_StorePackageFile", ctx, err)
		return
	}

	// Extract the parameter from the package
	params, err := util.GetParameters(pkgFile)
	if err != nil {
		util.HandleError("UploadPackage_ExtractParameter", ctx, err)
		return
	}
	pkg := &message.Package{Name: info.Filename, Parameters: params}

	err = a.resourceManager.GetPackageStore().CreatePackage(pkg)
	if err != nil {
		util.HandleError("UploadPackage_CreatePackage", ctx, err)
		return
	}
	ctx.JSON(pkg)
}

func (a APIHandler) GetTemplate(ctx iris.Context) {
	glog.Infof("Get template called")

	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		util.HandleError("GetTemplate_GetParam", ctx,
			util.NewInvalidInputError("The package ID is invalid.", err.Error()))
		return
	}
	pkg, err := a.resourceManager.GetPackageStore().GetPackage(uint(id))
	if err != nil {
		util.HandleError("GetTemplate_GetPackage", ctx, err)
		return
	}

	template, err := a.resourceManager.GetPackageManager().GetTemplate(pkg.Name)
	if err != nil {
		util.HandleError("GetTemplate_GetPackageFile", ctx, err)
		return
	}
	ctx.Write(template)
}

func (a APIHandler) ListPipelines(ctx iris.Context) {
	glog.Infof("List pipelines called")

	pipelines, err := a.resourceManager.GetPipelineStore().ListPipelines()
	if err != nil {
		util.HandleError("ListPipelines", ctx, err)
		return
	}

	ctx.JSON(pipelines)
}

func (a APIHandler) GetPipeline(ctx iris.Context) {
	glog.Infof("Get pipeline called")

	id, err := ctx.Params().GetInt64("pipelineId")
	if err != nil {
		util.HandleError("GetPipeline_GetParam", ctx, util.NewInvalidInputError("The pipeline ID is invalid.", err.Error()))
		return
	}
	pipeline, err := a.resourceManager.GetPipelineStore().GetPipeline(uint(id))

	if err != nil {
		util.HandleError("GetPipeline", ctx, err)
		return
	}

	ctx.JSON(pipeline)
}

func (a APIHandler) CreatePipeline(ctx iris.Context) {
	glog.Infof("Create pipeline called")

	pipeline := &message.Pipeline{}
	if err := ctx.ReadJSON(pipeline); err != nil {
		util.HandleError("CreatePipeline_ReadRequestBody",
			ctx, util.NewInvalidInputError("The pipeline has invalid format.", err.Error()))
		return
	}
	err, errPrefix := a.createPipelineInternal(pipeline)
	if err != nil {
		util.HandleError(errPrefix, ctx, err)
		return
	}

	ctx.JSON(pipeline)
}

func (a APIHandler) createPipelineInternal(pipeline *message.Pipeline) (error, string) {

	// Verify the package exists
	pkg, err := a.resourceManager.GetPackageStore().GetPackage(pipeline.PackageId)
	if err != nil {
		return err, "CreatePipeline_ValidPackageExist"
	}

	// If the pipeline runs on a schedule
	if pipeline.Schedule != "" {
		// Validate the pipeline schedule.
		_, err := cron.Parse(pipeline.Schedule)
		if err != nil {
			error := util.NewInvalidInputError(
				fmt.Sprintf("The pipeline schedule cannot be parsed: %s: %s", pipeline.Schedule, err),
				err.Error())
			return error, "CreatePipeline_ValidSchedule"
		}
	}

	// Create pipeline metadata
	err = a.resourceManager.GetPipelineStore().CreatePipeline(pipeline)
	if err != nil {
		return err, "CreatePipeline"
	}

	// If there is no pipeline schedule, the job is created immediately.
	if pipeline.Schedule == "" {

		template, err := a.resourceManager.GetPackageManager().GetTemplate(pkg.Name)
		if err != nil {
			return err, "CreatePipeline_GetPackageFile"
		}

		// Inject parameters user provided to the pipeline template.
		workflow, err := util.InjectParameters(template, pipeline.Parameters)
		if err != nil {
			return err, "CreatePipeline_CreateJob_InjectParameter"
		}

		_, err = a.resourceManager.GetJobStore().CreateJob(pipeline.ID, workflow,
			a.resourceManager.GetTime().Now().Unix())
		if err != nil {
			return err, "CreatePipeline_CreateJob"
		}
	}

	return nil, ""
}

func (a APIHandler) EnablePipeline(ctx iris.Context) {
	a.enablePipeline(ctx, true)
}

func (a APIHandler) DisablePipeline(ctx iris.Context) {
	a.enablePipeline(ctx, false)
}

func (a APIHandler) enablePipeline(ctx iris.Context, enabled bool) {
	glog.Infof("Enable pipeline")

	pipelineID, err := ctx.Params().GetInt64("pipelineId")
	if err != nil {
		util.NewUserError(
			errors.Wrap(err, "Error when parsing pipeline ID."),
			util.NewInvalidInputError("The pipeline ID is invalid.", err.Error())).
			PopulateContextAndLog(ctx)
		return
	}

	err = a.resourceManager.EnablePipeline(uint(pipelineID), enabled)
	if err != nil {
		util.PopulateContextAndLogError(ctx, err)
		return
	}
}


func (a APIHandler) ListJobs(ctx iris.Context) {
	glog.Infof("List jobs called")

	pipelineId, err := ctx.Params().GetInt64("pipelineId")
	if err != nil {
		util.HandleError("ListJobs_GetParam", ctx, util.NewInvalidInputError("The pipeline ID is invalid.", err.Error()))
		return
	}

	jobs, err := a.resourceManager.GetJobStore().ListJobs(uint(pipelineId))
	if err != nil {
		util.HandleError("ListJobs", ctx, err)
		return
	}

	ctx.JSON(jobs)
}

func (a APIHandler) GetJob(ctx iris.Context) {
	glog.Infof("Get job called")

	pipelineId, err := ctx.Params().GetInt64("pipelineId")
	if err != nil {
		util.HandleError("GetJob_GetParam", ctx, util.NewInvalidInputError("The pipeline ID is invalid.", err.Error()))
		return
	}

	jobName := ctx.Params().Get("jobName")

	job, err := a.resourceManager.GetJobStore().GetJob(uint(pipelineId), jobName)
	if err != nil {
		util.HandleError("GetJob", ctx, err)
		return
	}

	ctx.JSON(job)
}

func newApp(clientManager ClientManager) *iris.Application {
	
	apiHandler := &APIHandler{
		resourceManager: resource.NewResourceManager(
			clientManager.packageStore, clientManager.pipelineStore,
			clientManager.jobStore, clientManager.packageManager, clientManager.time),
	}

	app := iris.New()
	// registers a custom handler for 404 not found http (error) status code,
	// fires when route not found or manually by ctx.StatusCode(iris.StatusNotFound).
	app.OnErrorCode(iris.StatusNotFound, notFoundHandler)

	apiRouter := app.Party(apiRouterPrefix)

	// Packages
	apiRouter.Get(listPackages, apiHandler.ListPackages)
	apiRouter.Get(getPackage, apiHandler.GetPackage)
	apiRouter.Post(uploadPackage, apiHandler.UploadPackage)
	apiRouter.Get(getTemplate, apiHandler.GetTemplate)

	// Pipelines
	apiRouter.Get(listPipelines, apiHandler.ListPipelines)
	apiRouter.Get(getPipeline, apiHandler.GetPipeline)
	apiRouter.Post(createPipeline, apiHandler.CreatePipeline)
	apiRouter.Options(createPipeline, func(iris.Context) {})
	apiRouter.Post(enablePipeline, apiHandler.EnablePipeline)
	apiRouter.Post(disablePipeline, apiHandler.DisablePipeline)

	// Jobs
	apiRouter.Get(listJobs, apiHandler.ListJobs)
	apiRouter.Get(getJob, apiHandler.GetJob)
	return app
}

func notFoundHandler(ctx iris.Context) {
	ctx.HTML("Nothing is here.")
}
