package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/go-openapi/strfmt"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	crontab "github.com/robfig/cron"
)

type OutputFormat string

const (
	OutputFormatYaml OutputFormat = "yaml"
	OutputFormatJson OutputFormat = "json"
	OutputFormatGo   OutputFormat = "go"
)

const (
	endOfTime = math.MaxInt32 * 100
)

func PrettyPrintResult(writer io.Writer, outputFormat string, vs ...interface{}) {
	for _, v := range vs {
		if v != "" {
			PrettyPrint(writer, v, OutputFormat(outputFormat))
		}
	}
}

func PrettyPrintGo(writer io.Writer, v interface{}) {
	fmt.Fprintf(writer, "%+v\n", v)
}

func PrettyPrintJson(writer io.Writer, v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		PrettyPrintGo(writer, v)
	} else {
		fmt.Fprintln(writer, string(b))
	}
}

func PrettyPrintYaml(writer io.Writer, v interface{}) {
	b, err := yaml.Marshal(v)
	if err != nil {
		PrettyPrintGo(writer, v)
	} else {
		fmt.Fprintln(writer, string(b))
	}
}

func PrettyPrint(writer io.Writer, v interface{}, format OutputFormat) {
	switch format {
	case OutputFormatYaml:
		PrettyPrintYaml(writer, v)
	case OutputFormatJson:
		PrettyPrintJson(writer, v)
	case OutputFormatGo:
		PrettyPrintGo(writer, v)
	default:
		PrettyPrintJson(writer, v)
	}
}

func ValidateSingleString(args []string, argumentName string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("Missing '%s' argument", argumentName)
	}
	if len(args) > 1 {
		return "", fmt.Errorf("Too many arguments")
	}
	return args[0], nil
}

func ValidateArgumentCount(args []string, expectedCount int) ([]string, error) {
	if len(args) != expectedCount {
		return nil, fmt.Errorf("Expected %d arguments", expectedCount)
	}
	return args, nil
}

func ValidateInt64Min(value int64, minValue int64, flagName string) (int64, error) {
	if value < minValue {
		return 0, fmt.Errorf("Flag '%s' must be at least %d", flagName, minValue)
	}
	return value, nil
}

func ValidateStartTime(timeString string, flagName string, timeInterface util.TimeInterface) (
	*strfmt.DateTime, error) {
	if timeString == "" {
		return util.DateTimePointer(strfmt.DateTime(timeInterface.Now())), nil
	}
	return ValidateTime(timeString, flagName)
}

func ValidateEndTime(timeString string, flagName string) (
	*strfmt.DateTime, error) {
	if timeString == "" {
		return util.DateTimePointer(strfmt.DateTime(time.Unix(endOfTime, 0))), nil
	}
	return ValidateTime(timeString, flagName)
}

func ValidateTime(timeString string, flagName string) (*strfmt.DateTime, error) {
	if timeString == "" {
		return nil, nil
	}
	dateTime, err := strfmt.ParseDateTime(timeString)
	if err != nil {
		return nil, fmt.Errorf(
			"Value '%s' (flag '%s') is not a valid DateTime strfmt format (https://godoc.org/github.com/go-openapi/strfmt): %s",
			timeString, flagName, err.Error())
	}
	return &dateTime, err
}

func ValidateCron(cron string, flagName string) (*string, error) {
	if cron == "" {
		return nil, nil
	}
	_, err := crontab.Parse(cron)
	if err != nil {
		return nil, fmt.Errorf("Value '%s' (flag '%s') is not a valid cron schedule format (https://godoc.org/github.com/robfig/cron): %s",
			cron, flagName, err.Error())
	}
	return &cron, err
}

func ValidateDuration(period time.Duration) *int64 {
	if period == 0*time.Second {
		return nil
	}
	return util.Int64Pointer(int64(period.Seconds()))
}

func ValidateParams(params []string) (map[string]string, error) {
	result := make(map[string]string)
	if params == nil {
		return result, nil
	}
	for _, param := range params {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("Parameter format is not valid. Expected: 'NAME=VALUE'. Got: '%s'", param)
		}
		result[parts[0]] = parts[1]
	}
	return result, nil
}
