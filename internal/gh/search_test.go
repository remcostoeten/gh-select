package gh

import "testing"

func TestPlanSearch(t *testing.T) {
	const live = "heygen-com/liveavatar-gpt-live-demos"
	cases := []struct {
		in     string
		exact  string
		owner  string
		filter string
		query  string
	}{
		{in: ""},
		{in: "   "},
		{in: "react", owner: "react", query: "react in:name"},
		{in: "torvalds/", owner: "torvalds", query: "user:torvalds"},
		{in: "facebook/re", exact: "facebook/re", owner: "facebook", filter: "re", query: "re in:name user:facebook"},
		{in: "  spf13/cobra  ", exact: "spf13/cobra", owner: "spf13", filter: "cobra", query: "cobra in:name user:spf13"},
		{in: "/orphan", query: "orphan in:name"},
		{in: "spf13 cobra", exact: "spf13/cobra", owner: "spf13", filter: "cobra", query: "spf13 cobra"},
		{in: "github heygen liveav", exact: "heygen/liveav", owner: "heygen", filter: "liveav", query: "heygen liveav"},
		{in: "spf13-cobra", exact: "spf13/cobra", owner: "spf13-cobra", query: "spf13-cobra in:name"},
		{in: "react language:go", owner: "react", query: "react in:name language:go"},
	}
	for _, c := range cases {
		got := planSearch(c.in)
		first := ""
		if len(got.exact) > 0 {
			first = got.exact[0]
		}
		if first != c.exact || got.owner != c.owner || got.filter != c.filter || got.query != c.query {
			t.Errorf("planSearch(%q) = %+v, want exact %q owner %q filter %q query %q",
				c.in, got, c.exact, c.owner, c.filter, c.query)
		}
	}

	pasted := []string{
		"github.com/" + live,
		"github.com/" + live + ".git",
		"https://github.com/" + live,
		"http://github.com/" + live + "/",
		"Https://www.github.com/" + live,
		"www.github.com/" + live,
		"lbalbal://github.com/" + live,
		"github.nl/" + live,
		"https://github.co.uk/" + live + "/tree/main/src",
		"git@github.com:" + live + ".git",
		"git clone https://github.com/" + live + ".git",
		"git clone git@github.com:" + live + ".git my-dir",
		"https://github.com/" + live + "?tab=readme#install",
	}
	for _, in := range pasted {
		got := planSearch(in)
		if len(got.exact) != 1 || got.exact[0] != live {
			t.Errorf("planSearch(%q).exact = %v, want [%s]", in, got.exact, live)
		}
	}

	if got := planSearch("https://github.com/heygen-com"); got.owner != "heygen-com" || got.query != "user:heygen-com" {
		t.Errorf("owner URL = %+v, want owner heygen-com", got)
	}
	if got := planSearch("github/gitignore"); len(got.exact) != 1 || got.exact[0] != "github/gitignore" {
		t.Errorf("github owner = %+v, want exact github/gitignore", got)
	}
}

func TestHyphenGuesses(t *testing.T) {
	got := hyphenGuesses("heygen-com-liveavatar")
	if len(got) != 2 || got[0] != "heygen/com-liveavatar" || got[1] != "heygen-com/liveavatar" {
		t.Fatalf("hyphenGuesses = %v", got)
	}
	if got := hyphenGuesses("-a-"); len(got) != 0 {
		t.Fatalf("edge hyphens = %v, want none", got)
	}
}

func TestFilterByName(t *testing.T) {
	repos := []Repo{{NameWithOwner: "h/docs"}, {NameWithOwner: "h/liveavatar-gpt-live-demos"}}
	got := filterByName(repos, "live av")
	if len(got) != 1 || got[0].Name() != "liveavatar-gpt-live-demos" {
		t.Fatalf("filterByName = %v", got)
	}
	if got := filterByName(repos, ""); len(got) != 2 {
		t.Fatalf("empty filter = %v, want all", got)
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
