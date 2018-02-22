package main

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/googleprivate/ml/apiserver/src/dao"
	"github.com/kataras/iris"
)

const (
	listTemplates = "/templates"
	getTemplate   = "/templates/{id:string}"
)

type APIHandler struct {
	templateDao dao.TemplateDaoInterface
}

func (a APIHandler) ListTemplates(ctx iris.Context) {
	glog.Infof("List template called")

	templates, err := a.templateDao.ListTemplate()
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

func newApp(clientManager ClientManager) *iris.Application {
	apiHandler := APIHandler{
		templateDao: clientManager.templateDao,
	}
	app := iris.New()

	// registers a custom handler for 404 not found http (error) status code,
	// fires when route not found or manually by ctx.StatusCode(iris.StatusNotFound).
	app.OnErrorCode(iris.StatusNotFound, notFoundHandler)

	apiRouter := app.Party(apiRouterPrefix)
	apiRouter.Get(listTemplates, apiHandler.ListTemplates)
	apiRouter.Get(getTemplate, apiHandler.GetTemplate)
	return app
}

func notFoundHandler(ctx iris.Context) {
	ctx.HTML("Nothing is here.")
}
