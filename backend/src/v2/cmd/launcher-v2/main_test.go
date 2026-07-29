package main

import "testing"

func TestResolveNamespace(t *testing.T) {
	t.Run("prefers explicit namespace flag", func(t *testing.T) {
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

	t.Run("prefers NAMESPACE", func(t *testing.T) {
		t.Setenv("NAMESPACE", "kubeflow")
		t.Setenv("POD_NAMESPACE", "ignored")

		got, err := resolveNamespace("")
		if err != nil {
			t.Fatalf("resolveNamespace() error = %v", err)
		}
		if got != "kubeflow" {
			t.Fatalf("resolveNamespace() = %q, want %q", got, "kubeflow")
		}
	})

	t.Run("does not fall back to POD_NAMESPACE", func(t *testing.T) {
		t.Setenv("NAMESPACE", "")
		t.Setenv("POD_NAMESPACE", "kubeflow-from-pod")

		got, err := resolveNamespace("")
		if err == nil && got == "kubeflow-from-pod" {
			t.Fatal("resolveNamespace() fell back to POD_NAMESPACE")
		}
	})
}
