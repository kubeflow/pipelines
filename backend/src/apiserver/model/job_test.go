package model

import (
	"testing"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
)

func TestJob_UpdateWorkflow(t *testing.T) {
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "1",
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        false,
			MaxConcurrency: util.Int64Pointer(200),
			NoCatchup:      util.BoolPointer(true),
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{Name: "PARAM1", Value: "NEW_VALUE1"},
				},
				Spec: v1alpha1.WorkflowSpec{
					ServiceAccountName: "service-account",
				},
			},
			Trigger: swfapi.Trigger{
				CronSchedule: &swfapi.CronSchedule{
					StartTime: util.MetaV1TimePointer(metav1.NewTime(time.Unix(10, 0).UTC())),
					EndTime:   util.MetaV1TimePointer(metav1.NewTime(time.Unix(20, 0).UTC())),
					Cron:      "MY_CRON",
				},
				PeriodicSchedule: &swfapi.PeriodicSchedule{
					StartTime:      util.MetaV1TimePointer(metav1.NewTime(time.Unix(30, 0).UTC())),
					EndTime:        util.MetaV1TimePointer(metav1.NewTime(time.Unix(40, 0).UTC())),
					IntervalSecond: 50,
				},
			},
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Conditions: []swfapi.ScheduledWorkflowCondition{{
				Type:   swfapi.ScheduledWorkflowEnabled,
				Status: core.ConditionTrue,
			},
			},
		},
	})

	j := Job{UUID: "static-id"}
	err := j.UpdateWorkflow(swf)
	assert.Nil(t, err)

	expectedJob := Job{
		UUID:           "static-id",
		Name:           "MY_NAME",
		Namespace:      "MY_NAMESPACE",
		Enabled:        false,
		Conditions:     "Enabled",
		MaxConcurrency: 200,
		NoCatchup:      true,
		PipelineSpec: PipelineSpec{
			Parameters: "[{\"name\":\"PARAM1\",\"value\":\"NEW_VALUE1\"}]",
		},
		Trigger: Trigger{
			CronSchedule: CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(10),
				CronScheduleEndTimeInSec:   util.Int64Pointer(20),
				Cron:                       util.StringPointer("MY_CRON"),
			},
			PeriodicSchedule: PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(30),
				PeriodicScheduleEndTimeInSec:   util.Int64Pointer(40),
				IntervalSecond:                 util.Int64Pointer(50),
			},
		},
		ServiceAccount: "service-account",
	}

	assert.Equal(t, expectedJob, j)
}
