package logger

import (
	"fmt"
	"github.com/onsi/ginkgo/v2"
)

func Log(s string, arguments ...any) {
	ginkgo.GinkgoWriter.Println(fmt.Sprintf(s, arguments))
}
