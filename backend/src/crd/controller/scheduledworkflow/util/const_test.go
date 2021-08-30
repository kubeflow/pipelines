package util

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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
