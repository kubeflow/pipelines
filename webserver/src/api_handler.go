package main

import (
	"github.com/googleprivate/ml/webserver/src/dao"
	"github.com/googleprivate/ml/webserver/src/util"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

const (
	listTemplates = "/templates"
	getTemplate   = "/templates/{template}"
)

type APIHandler struct {
	templateDao dao.TemplateDaoInterface
}

func (a APIHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	log.Printf("get a list template call")

	templates, err := a.templateDao.ListTemplate()
	if err != nil {
		util.HandleError(w, err)
		return
	}

	util.HandleJSONPayload(w, templates)
	w.WriteHeader(http.StatusOK)
}

func (a APIHandler) GetTemplates(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// TODO(yangpa): Ignore the implementation. Use ORM to fetch data later
	template, err := a.templateDao.GetTemplate(vars["template"])
	if err != nil {
		util.HandleError(w, err)
		return
	}

	util.HandleJSONPayload(w, template)
	w.WriteHeader(http.StatusOK)
}

// Creates the restful Container and defines the routes the API will serve
func CreateRestAPIHandler(clientManager ClientManager) http.Handler {
	apiHandler := APIHandler{
		templateDao: dao.NewTemplateDao(clientManager.db),
	}

	route := mux.NewRouter()
	route.Path(listTemplates).Methods(http.MethodGet).HandlerFunc(apiHandler.ListTemplates)
	route.Path(getTemplate).Methods(http.MethodGet).HandlerFunc(apiHandler.GetTemplates)
	return route
}
