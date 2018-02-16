package dao

import (
	"database/sql"
	"errors"
	"github.com/googleprivate/ml/apiserver/src/message"
	"github.com/googleprivate/ml/apiserver/src/util"
)

type TemplateDaoInterface interface {
	ListTemplate() ([]message.Template, error)
	GetTemplate(templateId string) (message.Template, error)
}

type TemplateDao struct {
	db *sql.DB
}

func (dao *TemplateDao) ListTemplate() ([]message.Template, error) {
	// TODO(yangpa): Ignore the implementation. Use ORM to fetch data later
	rows, err := dao.db.Query("SELECT * FROM template")

	if err != nil {
		return nil, err
	}

	var templates []message.Template

	for rows.Next() {
		var t message.Template
		err = rows.Scan(&t.Id, &t.Description)
		util.CheckErr(err)
		templates = append(templates, t)
	}
	return templates, nil
}

func (dao *TemplateDao) GetTemplate(templateId string) (message.Template, error) {
	// TODO(yangpa): Ignore the implementation. Use ORM to fetch data later
	rows, err := dao.db.Query("SELECT * FROM template WHERE id=" + templateId)
	var t message.Template
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
