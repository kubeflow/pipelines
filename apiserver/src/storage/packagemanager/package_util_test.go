package packagemanager

import (
	"testing"
	"strings"
)

func TestCheckValidPackage(t *testing.T) {
	pkg := "apiVersion: argoproj.io/v1alpha1 \nkind: Workflow"
	err := checkValidPackage([]byte(pkg))
	if err != nil {
		t.Errorf("Fail to validate a valid package")
	}
}

func TestCheckValidPackageNoWorkflow(t *testing.T) {
	pkg := "apiVersion: argoproj.io/v1alpha1"
	err := checkValidPackage([]byte(pkg))
	if err == nil || !strings.Contains(err.Error(), "No workflow exist") {
		t.Errorf("Fail to validate a invalid package. Should complain no workflow exist.")
	}
}

func TestCheckValidPackageInvalidYAML(t *testing.T) {
	pkg := "I am invalid yaml"
	err := checkValidPackage([]byte(pkg))
	if err == nil || !strings.Contains(err.Error(), "invalid YAML format") {
		t.Errorf("Fail to validate a invalid package. Should complain invalid YAML format.")
	}
}
