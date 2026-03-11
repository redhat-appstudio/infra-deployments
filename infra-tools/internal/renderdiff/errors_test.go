package renderdiff

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestIsNotKustomizationError(t *testing.T) {
	tests := []struct {
		name   string
		errMsg string
		want   bool
	}{
		{
			name:   "exact krusty missing kustomization error",
			errMsg: "kustomize build /tmp/foo: unable to find one of 'kustomization.yaml', 'kustomization.yml' or 'Kustomization' in directory '/tmp/foo'",
			want:   true,
		},
		{
			name:   "wrapped missing kustomization error",
			errMsg: "building components/plain/staging on HEAD: kustomize build /tmp/foo: unable to find one of 'kustomization.yaml', 'kustomization.yml' or 'Kustomization' in directory '/tmp/foo'",
			want:   true,
		},
		{
			name:   "genuine build error",
			errMsg: "kustomize build /tmp/foo: accumulating resources: resource not found",
			want:   false,
		},
		{
			name:   "empty string",
			errMsg: "",
			want:   false,
		},
		{
			name:   "unrelated error",
			errMsg: "connection refused",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(IsNotKustomizationError(tt.errMsg)).To(Equal(tt.want))
		})
	}
}
