package logger

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
)

func Log(s string, arguments ...any) {
	formatedString := fmt.Sprintf(s, arguments...)
	ginkgo.GinkgoWriter.Println(formatedString)
}
