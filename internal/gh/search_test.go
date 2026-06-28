package gh

import "testing"

func TestBuildSearchQuery(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"   ", ""},
		{"react", "react in:name"},
		{"torvalds/", "user:torvalds"},
		{"facebook/re", "re in:name user:facebook"},
		{"  spf13/cobra  ", "cobra in:name user:spf13"},
		{"/orphan", "orphan in:name"},
	}
	for _, c := range cases {
		if got := buildSearchQuery(c.in); got != c.want {
			t.Errorf("buildSearchQuery(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHoistDefault(t *testing.T) {
	got := hoistDefault([]string{"a", "trunk", "b"}, "trunk")
	if len(got) != 3 || got[0] != "trunk" || got[1] != "a" || got[2] != "b" {
		t.Fatalf("hoist present = %v, want [trunk a b]", got)
	}
	if got := hoistDefault([]string{"a", "b"}, "missing"); len(got) != 2 || got[0] != "a" {
		t.Fatalf("hoist absent = %v, want [a b]", got)
	}
	if got := hoistDefault([]string{"a", "b"}, ""); len(got) != 2 || got[0] != "a" {
		t.Fatalf("hoist empty def = %v, want [a b]", got)
	}
}
