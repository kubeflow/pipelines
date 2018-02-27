package main

import (
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/storage"
	"ml/apiserver/src/storage/packagemanager"
	"ml/apiserver/src/util"

	"github.com/golang/glog"
	"github.com/kataras/iris"
	"github.com/rs/xid"
)

const (
	apiRouterPrefix = "/apis/v1alpha1"

	listPackages  = "/packages"
	getPackage    = "/packages/{id:string}"
	uploadPackage = "/packages/upload"
	getTemplate   = "/packages/{id:string}/templates"

	listJobs = "/pipelines/{id:string}/jobs"
)

type APIHandler struct {
	packageStore   storage.PackageStoreInterface
	jobStore       storage.JobStoreInterface
	packageManager packagemanager.PackageManagerInterface
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

	id := ctx.Params().Get("id")
	pkg, err := a.packageStore.GetPackage(id)

	if err != nil {
		util.HandleError("GetPackageFile", ctx, err)
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
	err = a.packageManager.CreatePackageFile(file, info)
	if err != nil {
		util.HandleError("UploadPackage", ctx, err)
		return
	}

	pkg := pipelinemanager.Package{Id: xid.New().String(), Name: info.Filename}
	err = a.packageStore.CreatePackage(pkg)
	if err != nil {
		util.HandleError("UploadPackage", ctx, err)
		return
	}
	ctx.JSON(pkg)
}

func (a APIHandler) GetTemplate(ctx iris.Context) {
	glog.Infof("Get template called")

	id := ctx.Params().Get("id")
	pkg, err := a.packageStore.GetPackage(id)
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

func (a APIHandler) ListJobs(ctx iris.Context) {
	glog.Infof("List jobs called")

	jobs, err := a.jobStore.ListJobs()
	if err != nil {
		util.HandleError("ListJobs", ctx, err)
		return
	}

	ctx.JSON(jobs)
}

func newApp(clientManager ClientManager) *iris.Application {
	apiHandler := APIHandler{
		packageStore:   clientManager.packageStore,
		jobStore:       clientManager.jobStore,
		packageManager: clientManager.packageManager,
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

	// Jobs
	apiRouter.Get(listJobs, apiHandler.ListJobs)
	return app
}

func notFoundHandler(ctx iris.Context) {
	ctx.HTML("Nothing is here.")
}
