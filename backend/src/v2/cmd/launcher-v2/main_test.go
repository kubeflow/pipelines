package main

import "testing"

func TestResolveStringFlag(t *testing.T) {
	t.Run("prefers primary flag", func(t *testing.T) {
		primary := "task-id"
		legacy := "legacy-task-id"
		if got := resolveStringFlag(&primary, &legacy); got != primary {
			t.Fatalf("resolveStringFlag() = %q, want %q", got, primary)
		}
	})

	t.Run("falls back to legacy flag", func(t *testing.T) {
		primary := ""
		legacy := "legacy-task-id"
		if got := resolveStringFlag(&primary, &legacy); got != legacy {
			t.Fatalf("resolveStringFlag() = %q, want %q", got, legacy)
		}
	})
}

func TestResolveTaskName(t *testing.T) {
	t.Run("prefers explicit task name", func(t *testing.T) {
		got, err := resolveTaskName("explicit-task", "")
		if err != nil {
			t.Fatalf("resolveTaskName() error = %v", err)
		}
		if got != "explicit-task" {
			t.Fatalf("resolveTaskName() = %q, want %q", got, "explicit-task")
		}
	})

	t.Run("falls back to legacy task spec", func(t *testing.T) {
		rawTaskSpec := `{"taskInfo":{"name":"legacy-task"}}`
		got, err := resolveTaskName("", rawTaskSpec)
		if err != nil {
			t.Fatalf("resolveTaskName() error = %v", err)
		}
		if got != "legacy-task" {
			t.Fatalf("resolveTaskName() = %q, want %q", got, "legacy-task")
		}
	})
}

func TestResolveNamespace(t *testing.T) {
	t.Run("prefers NAMESPACE", func(t *testing.T) {
		t.Setenv("NAMESPACE", "kubeflow")
		t.Setenv("POD_NAMESPACE", "ignored")

		got, err := resolveNamespace()
		if err != nil {
			t.Fatalf("resolveNamespace() error = %v", err)
		}
		if got != "kubeflow" {
			t.Fatalf("resolveNamespace() = %q, want %q", got, "kubeflow")
		}
	})

	t.Run("falls back to POD_NAMESPACE", func(t *testing.T) {
		t.Setenv("NAMESPACE", "")
		t.Setenv("POD_NAMESPACE", "kubeflow-from-pod")

		got, err := resolveNamespace()
		if err != nil {
			t.Fatalf("resolveNamespace() error = %v", err)
		}
		if got != "kubeflow-from-pod" {
			t.Fatalf("resolveNamespace() = %q, want %q", got, "kubeflow-from-pod")
		}
	})
}
