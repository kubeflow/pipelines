package dao

import (
	"database/sql"
	"errors"

	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"
)

type TemplateDaoInterface interface {
	ListTemplates() ([]pipelinemanager.Template, error)
	GetTemplate(templateId string) (pipelinemanager.Template, error)
}

type TemplateDao struct {
	db *sql.DB
}

func (dao *TemplateDao) ListTemplates() ([]pipelinemanager.Template, error) {
	// TODO(yangpa): Ignore the implementation. Use ORM to fetch data later
	rows, err := dao.db.Query("SELECT * FROM template")

	if err != nil {
		return nil, err
	}

	var templates []pipelinemanager.Template

	for rows.Next() {
		var t pipelinemanager.Template
		err = rows.Scan(&t.Id, &t.Description)
		util.CheckErr(err)
		templates = append(templates, t)
	}
	return templates, nil
}

func (dao *TemplateDao) GetTemplate(templateId string) (pipelinemanager.Template, error) {
	// TODO(yangpa): Ignore the implementation. Use ORM to fetch data later
	rows, err := dao.db.Query("SELECT * FROM template WHERE id=" + templateId)
	var t pipelinemanager.Template
	if err != nil {
		return t, err
	}

	if !rows.Next() {
		return t, errors.New("template not found")
	}

	err = rows.Scan(&t.Id, &t.Description)
	return t, err
}

// factory function for template DAO
func NewTemplateDao(db *sql.DB) *TemplateDao {
	return &TemplateDao{db: db}
}
