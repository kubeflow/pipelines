package main

import (
	"net/http"

	"ml/apiserver/src/dao"

	"github.com/golang/glog"
	"github.com/kataras/iris"
)

const (
	listTemplates = "/templates"
	getTemplate   = "/templates/{id:string}"
	// TODO runs should instead have resource path of /schedules/{id:string}/runs.
	listRuns = "/runs"
)

type APIHandler struct {
	templateDao dao.TemplateDaoInterface
	runDao      dao.RunDaoInterface
}

func (a APIHandler) ListTemplates(ctx iris.Context) {
	glog.Infof("List templates called")

	templates, err := a.templateDao.ListTemplates()
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	ctx.JSON(templates)
}

func (a APIHandler) GetTemplate(ctx iris.Context) {
	glog.Infof("Get template called")

	id := ctx.Params().Get("id")
	// TODO(yangpa): Ignore the implementation. Use ORM to fetch data later
	template, err := a.templateDao.GetTemplate(id)

	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	ctx.JSON(template)
}

func (a APIHandler) ListRuns(ctx iris.Context) {
	glog.Infof("List runs called")

	runs, err := a.runDao.ListRuns()
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	ctx.JSON(runs)
}

func newApp(clientManager ClientManager) *iris.Application {
	apiHandler := APIHandler{
		templateDao: clientManager.templateDao,
		runDao:      clientManager.runDao,
	}
	app := iris.New()

	// registers a custom handler for 404 not found http (error) status code,
	// fires when route not found or manually by ctx.StatusCode(iris.StatusNotFound).
	app.OnErrorCode(iris.StatusNotFound, notFoundHandler)

	apiRouter := app.Party(apiRouterPrefix)
	apiRouter.Get(listTemplates, apiHandler.ListTemplates)
	apiRouter.Get(getTemplate, apiHandler.GetTemplate)
	apiRouter.Get(listRuns, apiHandler.ListRuns)
	return app
}

func notFoundHandler(ctx iris.Context) {
	ctx.HTML("Nothing is here.")
}
