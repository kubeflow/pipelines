package util

import (
	"testing"
	"time"

	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLocationSet(t *testing.T) {
	locString := "Asia/Shanghai"
	viper.Set(TimeZone, locString)
	defer viper.Set(TimeZone, "")
	timezone, err := GetLocation()
	assert.Nil(t, err)
	expectedTimezone, _ := time.LoadLocation(locString)
	assert.Equal(t, expectedTimezone, timezone)
}

func TestGetLocationDefault(t *testing.T) {
	locString := "Local"
	timezone, err := GetLocation()
	assert.Nil(t, err)
	expectedTimezone, _ := time.LoadLocation(locString)
	assert.Equal(t, expectedTimezone, timezone)
}

// Check actual cron ticks across both DST transitions rather than comparing
// two locations loaded from the same timezone source.
func TestGetLocationNonUTCCronAcrossDST(t *testing.T) {
	viper.Set(TimeZone, "America/New_York")
	t.Cleanup(func() { viper.Set(TimeZone, "") })
	location, err := GetLocation()
	require.NoError(t, err)
	schedule := NewCronSchedule(&swfapi.CronSchedule{Cron: "0 0 9 * * *"})
	for _, tc := range []struct {
		name  string
		start string
		want  string
	}{
		{"spring", "2026-03-07T14:00:00Z", "2026-03-08T13:00:00Z"},
		{"fall", "2026-10-31T13:00:00Z", "2026-11-01T14:00:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start, err := time.Parse(time.RFC3339, tc.start)
			require.NoError(t, err)
			got := schedule.GetNextScheduledTime(nil, start, location)
			assert.Equal(t, tc.want, got.UTC().Format(time.RFC3339))
			assert.Equal(t, 9, got.Hour())
		})
	}
}

func TestGetLocationInvalidTimezone(t *testing.T) {
	viper.Set(TimeZone, "invalid/timezone")
	t.Cleanup(func() { viper.Set(TimeZone, "") })
	_, err := GetLocation()
	require.Error(t, err)
}
