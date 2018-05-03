package util

import (
	"regexp"
	"strings"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"
)

type WorkflowFormatter struct {
	uuid             UUIDGeneratorInterface
	scheduledAtInSec int64
	nowInSec         int64
}

func NewWorkflowFormatter(uuid UUIDGeneratorInterface, scheduledAtInSec int64,
	nowInSec int64) *WorkflowFormatter {

	if uuid == nil {
		glog.Fatalf("A UUID generator must be specified.") // Should never happen.
	}

	return &WorkflowFormatter{
		uuid:             uuid,
		scheduledAtInSec: scheduledAtInSec,
		nowInSec:         nowInSec,
	}
}

func (p *WorkflowFormatter) Format(workflow *v1alpha1.Workflow) error {
	p.fuseWorkflowNames(workflow)
	err := p.appendUUIDToWorkflowName(workflow)
	if err != nil {
		return err
	}
	err = p.formatWorkflowName(workflow)
	if err != nil {
		return err
	}
	err = p.formatWorkflowParameters(workflow)
	if err != nil {
		return err
	}
	return nil
}

func (p *WorkflowFormatter) formatWorkflowParameters(workflow *v1alpha1.Workflow) error {
	if workflow.Spec.Arguments.Parameters == nil {
		return nil
	}

	newParams := make([]v1alpha1.Parameter, 0)

	for _, param := range workflow.Spec.Arguments.Parameters {
		newParam, err := p.formatParameter(param)
		if err != nil {
			return err
		}
		newParams = append(newParams, *newParam)
	}

	workflow.Spec.Arguments.Parameters = newParams
	return nil
}

func (p *WorkflowFormatter) fuseWorkflowNames(workflow *v1alpha1.Workflow) {

	if workflow.Name != "" {
		workflow.GenerateName = ""
		return
	}

	if workflow.GenerateName != "" {
		workflow.Name = workflow.GenerateName
		workflow.GenerateName = ""
		return
	}
}

func (p *WorkflowFormatter) appendUUIDToWorkflowName(workflow *v1alpha1.Workflow) error {
	uuid, err := p.uuid.NewRandom()
	if err != nil {
		return err
	}

	if workflow.Name != "" {
		workflow.Name = workflow.Name + uuid.String()
		workflow.GenerateName = ""
		return nil
	}

	workflow.Name = uuid.String()
	return nil
}

func (p *WorkflowFormatter) formatWorkflowName(workflow *v1alpha1.Workflow) error {
	newName, err := p.formatString(workflow.Name)
	if err != nil {
		return err
	}
	workflow.Name = newName
	return nil
}

func (p *WorkflowFormatter) formatParameter(param v1alpha1.Parameter) (*v1alpha1.Parameter, error) {
	formatted, err := p.formatString(*param.Value)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.Parameter{
		Name:  param.Name,
		Value: &formatted,
	}, nil
}

func (p *WorkflowFormatter) formatString(s string) (string, error) {
	re := regexp.MustCompile("\\[\\[(.*?)\\]\\]")
	matches := re.FindAllString(s, -1)
	if matches == nil {
		return s, nil
	}

	result := s

	for _, match := range matches {
		substitute, err := p.createSubtitute(match)
		if err != nil {
			return "", err
		}
		result = strings.Replace(result, match, substitute, 1)
	}

	return result, nil
}

func (p *WorkflowFormatter) createSubtitute(match string) (string, error) {

	const (
		schedulePrefix            = "[[schedule."
		nowPrefix                 = "[[now."
		uuidExpression            = "[[uuid]]"
		defaultScheduleExpression = "[[schedule]]"
		defaultNowExpression      = "[[now]]"
		defaultTimeFormat         = "20060102150405"
		suffix                    = "]]"
	)

	if strings.HasPrefix(match, defaultScheduleExpression) {
		return time.Unix(p.scheduledAtInSec, 0).UTC().Format(defaultTimeFormat), nil
	} else if strings.HasPrefix(match, defaultNowExpression) {
		return time.Unix(p.nowInSec, 0).UTC().Format(defaultTimeFormat), nil
	} else if strings.HasPrefix(match, schedulePrefix) {
		match = strings.Replace(match, schedulePrefix, "", 1)
		match = strings.Replace(match, suffix, "", 1)
		return time.Unix(p.scheduledAtInSec, 0).UTC().Format(match), nil
	} else if strings.HasPrefix(match, nowPrefix) {
		match = strings.Replace(match, nowPrefix, "", 1)
		match = strings.Replace(match, suffix, "", 1)
		return time.Unix(p.nowInSec, 0).UTC().Format(match), nil
	} else if match == uuidExpression {
		uuid, err := p.uuid.NewRandom()
		if err != nil {
			return "", NewInternalServerError(err, "Could not generate UUID: %v", err.Error())
		}
		return uuid.String(), nil
	} else {
		return match, nil
	}
}

