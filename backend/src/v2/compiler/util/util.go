package util

import (
	"regexp"
	"strings"
)

func SanitizeK8sName(name string) string {
	expressionAlphaNumericOrHyphen := regexp.MustCompile(`[^-0-9a-z]+`)
	// replace non alphanumeric or hyphen characters to hyphen
	nameWithAlphaNumericAndHyphen :=
		expressionAlphaNumericOrHyphen.ReplaceAll(
			[]byte(strings.ToLower(name)),
			[]byte("-"))
	expressionMultipleHyphens := regexp.MustCompile(`-+`)
	// compress multiple hyphens to one
	return string(expressionMultipleHyphens.ReplaceAll(
		nameWithAlphaNumericAndHyphen,
		[]byte("-")))
}
