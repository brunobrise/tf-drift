package drift

import "testing"

func TestHomePathForDisplayUsesTildeForHome(t *testing.T) {
	tests := []struct {
		name string
		path string
		home string
		want string
	}{
		{
			name: "exact home",
			path: "/Users/alice",
			home: "/Users/alice",
			want: "~",
		},
		{
			name: "home child",
			path: "/Users/alice/code/repo",
			home: "/Users/alice",
			want: "~/code/repo",
		},
		{
			name: "not sibling prefix",
			path: "/Users/alice-other/code",
			home: "/Users/alice",
			want: "/Users/alice-other/code",
		},
		{
			name: "root home",
			path: "/root/project",
			home: "/root",
			want: "~/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := homePathForDisplay(tt.path, tt.home); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestSelectionDisplayPathFallsBackToTildePath(t *testing.T) {
	got := selectionDisplayPath("/tmp/base", "/Users/alice/project/layer", "/Users/alice")
	if got != "~/project/layer" {
		t.Fatalf("expected tilde path, got %q", got)
	}
}
