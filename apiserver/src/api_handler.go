package main

import (
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/storage"
	"ml/apiserver/src/util"

	"github.com/golang/glog"
	"github.com/kataras/iris"
	"github.com/ghodss/yaml"
	"bytes"
	"io"
	"github.com/iris-contrib/middleware/cors"
)

const (
	apiRouterPrefix = "/apis/v1alpha1"

	listPackages  = "/packages"
	getPackage    = "/packages/{id:long min(1)}"
	uploadPackage = "/packages/upload"
	getTemplate   = "/packages/{id:long min(1)}/templates"

	listPipelines  = "/pipelines"
	getPipeline    = "/pipelines/{pipelineId:long min(1)}"
	createPipeline = "/pipelines"

	listJobs  = "/pipelines/{pipelineId:long min(1)}/jobs"
	createJob = "/pipelines/{pipelineId:long min(1)}/jobs"
)

type APIHandler struct {
	packageStore   storage.PackageStoreInterface
	pipelineStore  storage.PipelineStoreInterface
	jobStore       storage.JobStoreInterface
	packageManager storage.PackageManagerInterface
}

func (a APIHandler) ListPackages(ctx iris.Context) {
	glog.Infof("List packages called")

	packages, err := a.packageStore.ListPackages()
	if err != nil {
		util.HandleError("ListPackages", ctx, err)
		return
	}

	ctx.JSON(packages)
}

func (a APIHandler) GetPackage(ctx iris.Context) {
	glog.Infof("Get package called")

	id, err := ctx.Params().GetInt64("id")
	if err != nil || id < 0 {
		util.HandleError("GetPackage", ctx, util.NewInvalidInputError("The package ID is invalid."))
	}
	pkg, err := a.packageStore.GetPackage(uint(id))

	if err != nil {
		util.HandleError("GetPackage", ctx, err)
		return
	}

	ctx.JSON(pkg)
}

func (a APIHandler) UploadPackage(ctx iris.Context) {
	glog.Infof("Upload package called")

	// Get the file from the request.
	file, info, err := ctx.FormFile("uploadfile")

	if err != nil {
		util.HandleError("UploadPackage", ctx, err)
		return
	}

	defer file.Close()

	// We stream the file to API server for now. This is OK for now since it's just a small YAML file.
	// In near future, use Minio Presigned Put instead.
	// For more info check https://docs.minio.io/docs/golang-client-api-reference#PresignedPutObject
	buf := bytes.NewBuffer(nil)
	if _, err = io.Copy(buf, file); err != nil {
		util.HandleError("UploadPackage", ctx, util.NewInternalError("Failed to copy package.", err.Error()))
		return
	}

	template := buf.Bytes()
	err = a.packageManager.CreatePackageFile(template, info.Filename)
	if err != nil {
		util.HandleError("UploadPackage", ctx, err)
		return
	}

	params, err := util.GetParameter(template)
	if err != nil {
		util.HandleError("UploadPackage", ctx, err)
		return
	}
	pkg := pipelinemanager.Package{Name: info.Filename, Parameters: params}

	pkg, err = a.packageStore.CreatePackage(pkg)
	if err != nil {
		util.HandleError("UploadPackage", ctx, err)
		return
	}
	ctx.JSON(pkg)
}

func (a APIHandler) GetTemplate(ctx iris.Context) {
	glog.Infof("Get template called")

	id, err := ctx.Params().GetInt64("id")
	if err != nil || id < 0 {
		util.HandleError("GetPackage", ctx, util.NewInvalidInputError("The package ID is invalid."))
	}
	pkg, err := a.packageStore.GetPackage(uint(id))
	if err != nil {
		util.HandleError("GetTemplate", ctx, err)
		return
	}

	file, err := a.packageManager.GetPackageFile(pkg.Name)
	if err != nil {
		util.HandleError("GetTemplate", ctx, err)
		return
	}
	ctx.Write(file)
}

func (a APIHandler) ListPipelines(ctx iris.Context) {
	glog.Infof("List pipelines called")

	pipelines, err := a.pipelineStore.ListPipelines()
	if err != nil {
		util.HandleError("ListPipelines", ctx, err)
		return
	}

	ctx.JSON(pipelines)
}

func (a APIHandler) GetPipeline(ctx iris.Context) {
	glog.Infof("Get pipeline called")

	id, err := ctx.Params().GetInt64("pipelineId")
	if err != nil || id < 0 {
		util.HandleError("GetPipeline", ctx, util.NewInvalidInputError("The pipeline ID is invalid."))
	}
	pipeline, err := a.pipelineStore.GetPipeline(uint(id))

	if err != nil {
		util.HandleError("GetPipeline", ctx, err)
		return
	}

	ctx.JSON(pipeline)
}

func (a APIHandler) CreatePipeline(ctx iris.Context) {
	glog.Infof("Create pipeline called")

	pipeline := pipelinemanager.Pipeline{}
	if err := ctx.ReadJSON(&pipeline); err != nil {
		util.HandleError("CreatePipeline", ctx, util.NewInvalidInputError(err.Error()))
		return
	}

	// Verify the package exist
	_, err := a.packageStore.GetPackage(pipeline.PackageId)
	if err != nil {
		util.HandleError("CreatePipeline", ctx, err)
		return
	}

	pipeline, err = a.pipelineStore.CreatePipeline(pipeline)
	if err != nil {
		util.HandleError("CreatePipeline", ctx, err)
		return
	}

	ctx.JSON(pipeline)
}

func (a APIHandler) ListJobs(ctx iris.Context) {
	glog.Infof("List jobs called")

	jobs, err := a.jobStore.ListJobs()
	if err != nil {
		util.HandleError("ListJobs", ctx, err)
		return
	}

	ctx.JSON(jobs)
}

func (a APIHandler) CreateJob(ctx iris.Context) {
	glog.Infof("Create jobs called")

	id, err := ctx.Params().GetInt64("pipelineId")
	if err != nil || id < 0 {
		util.HandleError("GetPipeline", ctx, util.NewInvalidInputError("The pipeline ID is invalid."))
	}
	pipeline, err := a.pipelineStore.GetPipeline(uint(id))

	if err != nil {
		util.HandleError("CreateJob", ctx, err)
		return
	}

	pkg, err := a.packageStore.GetPackage(pipeline.PackageId)
	if err != nil {
		util.HandleError("GetPackageFile", ctx, err)
		return
	}

	file, err := a.packageManager.GetPackageFile(pkg.Name)
	if err != nil {
		util.HandleError("GetTemplate", ctx, err)
		return
	}

	file, err = util.InjectParameter(file, pipeline.Parameters)
	if err != nil {
		util.HandleError("GetTemplate", ctx, err)
		return
	}

	file, err = yaml.YAMLToJSON(file)
	if err != nil {
		util.HandleError("GetTemplate", ctx, err)
		return
	}

	job, err := a.jobStore.CreateJob(file)
	if err != nil {
		util.HandleError("CreateJob", ctx, err)
		return
	}

	ctx.JSON(job)
}

func newApp(clientManager ClientManager) *iris.Application {
	apiHandler := APIHandler{
		packageStore:   clientManager.packageStore,
		pipelineStore:  clientManager.pipelineStore,
		jobStore:       clientManager.jobStore,
		packageManager: clientManager.packageManager,
	}
	app := iris.New()
	// Allow all domains for now.
	app.Use(cors.NewAllowAll())
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

	// Jobs
	apiRouter.Get(listJobs, apiHandler.ListJobs)
	apiRouter.Post(createJob, apiHandler.CreateJob)
	apiRouter.Options(createJob, func(iris.Context) {})
	return app
}

func notFoundHandler(ctx iris.Context) {
	ctx.HTML("Nothing is here.")
}
