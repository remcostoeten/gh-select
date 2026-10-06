package sys

import (
	"reflect"
	"testing"
)

func TestResolveEditor(t *testing.T) {
	cases := []struct {
		name, visual, editor string
		wantName             string
		wantArgs             []string
	}{
		{"editor with flags", "", "code -w", "code", []string{"-w"}},
		{"visual wins over editor", "zed", "vim", "zed", []string{}},
		{"blank visual falls through", "   ", "nvim", "nvim", []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VISUAL", tc.visual)
			t.Setenv("EDITOR", tc.editor)
			name, args := resolveEditor()
			if name != tc.wantName || !reflect.DeepEqual(args, tc.wantArgs) {
				t.Errorf("got %q %q, want %q %q", name, args, tc.wantName, tc.wantArgs)
			}
		})
	}
}
