// +build tools

package tools

// This file tracks tools used in the project, but not directly imported by code.
// https://github.com/go-modules-by-example/index/blob/master/010_tools/README.md

import (
	_ "github.com/google/addlicense"
	_ "k8s.io/code-generator"
)
