package securitycontext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRunAsNonRootEffective(t *testing.T) {
	tr := true
	fa := false

	tests := []struct {
		name      string
		admin     *bool
		component *bool
		want      bool
	}{
		{"both nil", nil, nil, false},
		{"admin true", &tr, nil, true},
		{"admin false", &fa, nil, false},
		{"component true", nil, &tr, true},
		{"component false", nil, &fa, false},
		{"admin true overrides component false", &tr, &fa, true},
		{"admin false overrides component true", &fa, &tr, false},
		{"both true", &tr, &tr, true},
		{"both false", &fa, &fa, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRunAsNonRootEffective(tt.admin, tt.component))
		})
	}
}
