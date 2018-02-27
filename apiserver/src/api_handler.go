package main

import (
	"ml/apiserver/src/dao"
	"ml/apiserver/src/util"

	"github.com/golang/glog"
	"github.com/kataras/iris"
)

const (
	apiRouterPrefix = "/apis/v1alpha1"

	listPackages = "/packages"
	getPackage   = "/packages/{id:string}"
	listJobs     = "/pipelines/{id:string}/jobs"
)

type APIHandler struct {
	packageDao dao.PackageDaoInterface
	jobDao     dao.JobDaoInterface
}

func (a APIHandler) ListPackages(ctx iris.Context) {
	glog.Infof("List packages called")

	packages, err := a.packageDao.ListPackages()
	if err != nil {
		util.HandleError("ListPackages", ctx, err)
		return
	}

	ctx.JSON(packages)
}

func (a APIHandler) GetPackage(ctx iris.Context) {
	glog.Infof("Get package called")

	id := ctx.Params().Get("id")
	// TODO(yangpa): Ignore the implementation. Use ORM to fetch data later
	pkg, err := a.packageDao.GetPackage(id)

	if err != nil {
		util.HandleError("GetPackage", ctx, err)
		return
	}

	ctx.JSON(pkg)
}

func (a APIHandler) ListJobs(ctx iris.Context) {
	glog.Infof("List jobs called")

	jobs, err := a.jobDao.ListJobs()
	if err != nil {
		util.HandleError("ListJobs", ctx, err)
		return
	}

	ctx.JSON(jobs)
}

func newApp(clientManager ClientManager) *iris.Application {
	apiHandler := APIHandler{
		packageDao: clientManager.packageDao,
		jobDao:     clientManager.jobDao,
	}
	app := iris.New()

	// registers a custom handler for 404 not found http (error) status code,
	// fires when route not found or manually by ctx.StatusCode(iris.StatusNotFound).
	app.OnErrorCode(iris.StatusNotFound, notFoundHandler)

	apiRouter := app.Party(apiRouterPrefix)
	apiRouter.Get(listPackages, apiHandler.ListPackages)
	apiRouter.Get(getPackage, apiHandler.GetPackage)
	apiRouter.Get(listJobs, apiHandler.ListJobs)
	return app
}

func notFoundHandler(ctx iris.Context) {
	ctx.HTML("Nothing is here.")
}
