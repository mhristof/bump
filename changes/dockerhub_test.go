package changes

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDockerHub(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  string
	}{
		{
			name:  "valid docker image without https:// prefix",
			image: "prom/alertmanager:v0.25.0",
			want:  "^prom/alertmanager:v.*$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Regexp(t, regexp.MustCompile(tt.want), dockerHub(tt.image))
		})
	}
}
