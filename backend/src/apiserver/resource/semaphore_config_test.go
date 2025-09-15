package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Simple unit tests for semaphore configuration parsing
func TestExtractSemaphoreConfig_ValidSemaphoreKey(t *testing.T) {
	manager := &ResourceManager{}

	manifest := `{
		"platformSpec": {
			"platforms": {
				"kubernetes": {
					"pipelineConfig": {
						"semaphoreKey": "gpu-training"
					}
				}
			}
		}
	}`

	semaphoreKey, mutexName, err := manager.extractSemaphoreConfig(manifest)

	assert.NoError(t, err)
	assert.Equal(t, "gpu-training", semaphoreKey)
	assert.Equal(t, "", mutexName)
}

func TestExtractSemaphoreConfig_ValidMutexName(t *testing.T) {
	manager := &ResourceManager{}

	manifest := `{
		"platformSpec": {
			"platforms": {
				"kubernetes": {
					"pipelineConfig": {
						"mutexName": "data-processor"
					}
				}
			}
		}
	}`

	semaphoreKey, mutexName, err := manager.extractSemaphoreConfig(manifest)

	assert.NoError(t, err)
	assert.Equal(t, "", semaphoreKey)
	assert.Equal(t, "data-processor", mutexName)
}

func TestExtractSemaphoreConfig_BothFields(t *testing.T) {
	manager := &ResourceManager{}

	manifest := `{
		"platformSpec": {
			"platforms": {
				"kubernetes": {
					"pipelineConfig": {
						"semaphoreKey": "gpu-training",
						"mutexName": "data-processor"
					}
				}
			}
		}
	}`

	semaphoreKey, mutexName, err := manager.extractSemaphoreConfig(manifest)

	assert.NoError(t, err)
	assert.Equal(t, "gpu-training", semaphoreKey)
	assert.Equal(t, "data-processor", mutexName)
}

func TestExtractSemaphoreConfig_NoPipelineConfig(t *testing.T) {
	manager := &ResourceManager{}

	manifest := `{
		"platformSpec": {
			"platforms": {
				"kubernetes": {
					"someOtherField": "value"
				}
			}
		}
	}`

	semaphoreKey, mutexName, err := manager.extractSemaphoreConfig(manifest)

	assert.NoError(t, err)
	assert.Equal(t, "", semaphoreKey)
	assert.Equal(t, "", mutexName)
}

func TestExtractSemaphoreConfig_NoKubernetes(t *testing.T) {
	manager := &ResourceManager{}

	manifest := `{
		"platformSpec": {
			"platforms": {
				"someOtherPlatform": {
					"config": "value"
				}
			}
		}
	}`

	semaphoreKey, mutexName, err := manager.extractSemaphoreConfig(manifest)

	assert.NoError(t, err)
	assert.Equal(t, "", semaphoreKey)
	assert.Equal(t, "", mutexName)
}

func TestExtractSemaphoreConfig_NoPlatformSpec(t *testing.T) {
	manager := &ResourceManager{}

	manifest := `{
		"pipelineSpec": {
			"root": {
				"dag": {}
			}
		}
	}`

	semaphoreKey, mutexName, err := manager.extractSemaphoreConfig(manifest)

	assert.NoError(t, err)
	assert.Equal(t, "", semaphoreKey)
	assert.Equal(t, "", mutexName)
}

func TestExtractSemaphoreConfig_InvalidJSON(t *testing.T) {
	manager := &ResourceManager{}

	manifest := `{ unclosed bracket`

	semaphoreKey, mutexName, err := manager.extractSemaphoreConfig(manifest)

	assert.Error(t, err)
	assert.Equal(t, "", semaphoreKey)
	assert.Equal(t, "", mutexName)
}

func TestExtractSemaphoreConfig_EmptyManifest(t *testing.T) {
	manager := &ResourceManager{}

	manifest := `{}`

	semaphoreKey, mutexName, err := manager.extractSemaphoreConfig(manifest)

	assert.NoError(t, err)
	assert.Equal(t, "", semaphoreKey)
	assert.Equal(t, "", mutexName)
}

func TestExtractSemaphoreConfig_RealWorldManifest(t *testing.T) {
	manager := &ResourceManager{}

	// Simulates a real KFP v2 pipeline manifest
	manifest := `{
		"pipelineInfo": {
			"name": "test-pipeline"
		},
		"root": {
			"dag": {
				"tasks": {
					"task1": {
						"componentRef": {
							"name": "test-component"
						}
					}
				}
			}
		},
		"platformSpec": {
			"platforms": {
				"kubernetes": {
					"pipelineConfig": {
						"semaphoreKey": "gpu-cluster",
						"mutexName": "critical-resource"
					}
				}
			}
		},
		"components": {
			"test-component": {
				"executorLabel": "test-executor"
			}
		}
	}`

	semaphoreKey, mutexName, err := manager.extractSemaphoreConfig(manifest)

	assert.NoError(t, err)
	assert.Equal(t, "gpu-cluster", semaphoreKey)
	assert.Equal(t, "critical-resource", mutexName)
}

func TestSemaphoreManager_GetSemaphoreConfigMapName_Default(t *testing.T) {
	manager := &SemaphoreManager{}

	name := manager.getSemaphoreConfigMapName()

	assert.Equal(t, SemaphoreConfigName, name)
}

// Test with various edge cases for semaphore keys
func TestExtractSemaphoreConfig_EdgeCases(t *testing.T) {
	testCases := []struct {
		name              string
		manifest          string
		expectedSemaphore string
		expectedMutex     string
		shouldError       bool
	}{
		{
			name: "Empty string values",
			manifest: `{
				"platformSpec": {
					"platforms": {
						"kubernetes": {
							"pipelineConfig": {
								"semaphoreKey": "",
								"mutexName": ""
							}
						}
					}
				}
			}`,
			expectedSemaphore: "",
			expectedMutex:     "",
			shouldError:       false,
		},
		{
			name: "Special characters in keys",
			manifest: `{
				"platformSpec": {
					"platforms": {
						"kubernetes": {
							"pipelineConfig": {
								"semaphoreKey": "gpu-training-v2",
								"mutexName": "critical_resource_lock"
							}
						}
					}
				}
			}`,
			expectedSemaphore: "gpu-training-v2",
			expectedMutex:     "critical_resource_lock",
			shouldError:       false,
		},
		{
			name: "Numeric values (should still be parsed as strings)",
			manifest: `{
				"platformSpec": {
					"platforms": {
						"kubernetes": {
							"pipelineConfig": {
								"semaphoreKey": "123",
								"mutexName": "456"
							}
						}
					}
				}
			}`,
			expectedSemaphore: "123",
			expectedMutex:     "456",
			shouldError:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := &ResourceManager{}

			semaphoreKey, mutexName, err := manager.extractSemaphoreConfig(tc.manifest)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedSemaphore, semaphoreKey)
				assert.Equal(t, tc.expectedMutex, mutexName)
			}
		})
	}
}

func TestExtractSemaphoreConfig_UserPipelineFormat(t *testing.T) {
	tests := []struct {
		name            string
		manifest        string
		expectedSemKey  string
		expectedMutName string
		expectError     bool
	}{
		{
			name: "User's actual pipeline format - platforms at root",
			manifest: `# PIPELINE DEFINITION
components:
  comp-hello-world:
    executorLabel: exec-hello-world
pipelineInfo:
  description: A simple intro pipeline
  name: hello-world
root:
  dag:
    tasks:
      hello-world:
        componentRef:
          name: comp-hello-world
---
platforms:
  kubernetes:
    pipelineConfig:
      mutexName: mutex
      semaphoreKey: semaphore`,
			expectedSemKey:  "semaphore",
			expectedMutName: "mutex",
			expectError:     false,
		},
	}

	manager := &ResourceManager{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			semKey, mutName, err := manager.extractSemaphoreConfig(tt.manifest)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedSemKey, semKey, "semaphoreKey mismatch")
			assert.Equal(t, tt.expectedMutName, mutName, "mutexName mismatch")
		})
	}
}
