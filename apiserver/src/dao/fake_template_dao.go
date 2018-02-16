package dao

import (
	"errors"
	"github.com/googleprivate/ml/webserver/src/message"
)

type FakeTemplateDao struct {
}

func (dao *FakeTemplateDao) ListTemplate() ([]message.Template, error) {
	templates := []message.Template{
		{Id: 123, Description: "first description"},
		{Id: 456, Description: "second description"}}
	return templates, nil
}

func (dao *FakeTemplateDao) GetTemplate(templateId string) (message.Template, error) {
	template := message.Template{Id: 123, Description: "first description"}
	return template, nil
}

type FakeBadTemplateDao struct {
}

func (dao *FakeBadTemplateDao) ListTemplate() ([]message.Template, error) {
	return nil, errors.New("there is no template here")
}

func (dao *FakeBadTemplateDao) GetTemplate(templateId string) (message.Template, error) {
	return message.Template{}, errors.New("there is no template here")
}
