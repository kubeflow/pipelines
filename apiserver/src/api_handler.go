package main

import (
	"io/ioutil"
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/storage"
	"ml/apiserver/src/util"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris"
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
	if err != nil {
		util.HandleError("GetPackage_GetParam", ctx, util.NewInvalidInputError("The package ID is invalid.", err.Error()))
		return
	}
	pkg, err := a.packageStore.GetPackage(uint(id))

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
	err = a.packageManager.CreatePackageFile(pkgFile, info.Filename)
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
	pkg := pipelinemanager.Package{Name: info.Filename, Parameters: params}

	pkg, err = a.packageStore.CreatePackage(pkg)
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
		util.HandleError("GetTemplate_GetParam", ctx, util.NewInvalidInputError("The package ID is invalid.", err.Error()))
		return
	}
	pkg, err := a.packageStore.GetPackage(uint(id))
	if err != nil {
		util.HandleError("GetTemplate_GetPackage", ctx, err)
		return
	}

	file, err := a.packageManager.GetPackageFile(pkg.Name)
	if err != nil {
		util.HandleError("GetTemplate_GetPackageFile", ctx, err)
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
	if err != nil {
		util.HandleError("GetPipeline_GetParam", ctx, util.NewInvalidInputError("The pipeline ID is invalid.", err.Error()))
		return
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
		util.HandleError("CreatePipeline_ReadRequestBody", ctx, util.NewInvalidInputError("The pipeline has invalid format.", err.Error()))
		return
	}

	// Verify the package exist
	_, err := a.packageStore.GetPackage(pipeline.PackageId)
	if err != nil {
		util.HandleError("CreatePipeline_ValidPackageExist", ctx, err)
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

	_, err := ctx.Params().GetInt64("pipelineId")
	if err != nil {
		util.HandleError("ListJobs_GetParam", ctx, util.NewInvalidInputError("The pipeline ID is invalid.", err.Error()))
		return
	}

	jobs, err := a.jobStore.ListJobs()
	if err != nil {
		util.HandleError("ListJobs", ctx, err)
		return
	}

	ctx.JSON(jobs)
}

func (a APIHandler) CreateJob(ctx iris.Context) {
	glog.Infof("Create job called")

	id, err := ctx.Params().GetInt64("pipelineId")
	if err != nil {
		util.HandleError("CreateJob_GetParam", ctx, util.NewInvalidInputError("The pipeline ID is invalid.", err.Error()))
		return
	}

	// Get the pipeline metadata from the DB
	pipeline, err := a.pipelineStore.GetPipeline(uint(id))
	if err != nil {
		util.HandleError("CreateJob_GetPipeline", ctx, err)
		return
	}

	// Get the package metadata from the DB
	pkg, err := a.packageStore.GetPackage(pipeline.PackageId)
	if err != nil {
		util.HandleError("CreateJob_GetPackage", ctx, err)
		return
	}

	// Retrieve the actual package file.
	file, err := a.packageManager.GetPackageFile(pkg.Name)
	if err != nil {
		util.HandleError("CreateJob_GetPackageFile", ctx, err)
		return
	}

	// Inject parameters user provided to the pipeline template.
	file, err = util.InjectParameters(file, pipeline.Parameters)
	if err != nil {
		util.HandleError("CreateJob_InjectParameter", ctx, err)
		return
	}

	// Convert pipeline definition to Json format
	file, err = yaml.YAMLToJSON(file)
	if err != nil {
		util.HandleError("CreateJob_ConvertToJson", ctx, err)
		return
	}

	// Call K8s to create a new Argo workflow.
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
	// TODO(yangpa): Limit the origins to only allow webserver.
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
