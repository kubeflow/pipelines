package main

import "testing"

func TestResolveNamespace(t *testing.T) {
	t.Run("requires explicit namespace flag", func(t *testing.T) {
		t.Setenv("NAMESPACE", "ignored")
		t.Setenv("POD_NAMESPACE", "ignored-too")

		got, err := resolveNamespace("flag-namespace")
		if err != nil {
			t.Fatalf("resolveNamespace() error = %v", err)
		}
		if got != "flag-namespace" {
			t.Fatalf("resolveNamespace() = %q, want %q", got, "flag-namespace")
		}
	})

	t.Run("fails when namespace flag is missing", func(t *testing.T) {
		t.Setenv("NAMESPACE", "kubeflow")

		got, err := resolveNamespace("")
		if err == nil {
			t.Fatalf("resolveNamespace() = %q, want error", got)
		}
	})
}
