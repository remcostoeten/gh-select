package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/remcostoeten/gh-select/internal/ui"
	"golang.org/x/term"
)

// command is a subcommand reachable as a word, a short flag and a long flag,
// so `gh select limits`, `gh select -l` and `gh select --limits` all agree.
type command struct {
	name    string
	short   string
	args    string
	summary string
	details []string
}

var commands = []command{
	{
		name:    "doctor",
		summary: "check git, gh and your GitHub login",
		details: []string{
			"Checks that git and gh are on your PATH and that gh is logged in.",
			"It needs no login itself, so it still runs when everything else fails.",
		},
	},
	{
		name:    "limits",
		short:   "-l",
		summary: "show GitHub API rate limit usage",
		details: []string{
			"Prints the search, core, graphql and code search budgets, which every",
			"tool using your gh login shares. Each bar marks how far the current",
			"window has run and, when the current pace would empty a bucket before",
			"it resets, how soon that happens. The ? overlay in the app shows the same.",
		},
	},
	{
		name:    "refresh",
		short:   "-r",
		summary: "refetch the repo cache and exit",
		details: []string{
			"Fetches every repository you can access, saves it to the cache and",
			"prints how many it found. The app already refreshes a stale cache in",
			"the background, so this is for cron jobs and scripts.",
		},
	},
	{
		name:    "version",
		short:   "-v",
		summary: "print the installed version",
		details: []string{
			"Prints the release version. Local builds print the commit instead,",
			"like dev+5006ea0, with -dirty when the tree had uncommitted changes.",
		},
	},
	{
		name:    "help",
		short:   "-h",
		args:    " [command]",
		summary: "show this help, or one command's",
		details: []string{
			"Without a command it prints the overview. With one it prints that",
			"command's help, as does adding -h or --help to any command.",
		},
	},
}

// errUsage marks a command line that couldn't be understood; main prints it
// with a pointer to the help and exits 2.
type errUsage struct {
	msg string
}

func (e errUsage) Error() string { return e.msg }

// errCancelled ends a --print-path run that picked nothing, so a shell wrapper
// like `dir=$(gh select -p) && cd "$dir"` doesn't cd into an empty string.
var errCancelled = errors.New("cancelled")

func findCommand(token string) (command, bool) {
	for _, c := range commands {
		if token == c.name || token == "--"+c.name || (c.short != "" && token == c.short) {
			return c, true
		}
	}
	return command{}, false
}

// invocation is the routed command line: a command, whether its help was
// asked for, and whatever is left for the picker's own options.
type invocation struct {
	command string
	help    bool
	rest    []string
}

// route pulls the command out of the arguments. Flag forms count anywhere, but
// a bare word only counts first (or right after `help`), so a value such as
// `--dir refresh` is never mistaken for a command.
func route(args []string) (invocation, error) {
	var inv invocation
	for i, arg := range args {
		c, ok := findCommand(arg)
		bare := !strings.HasPrefix(arg, "-")
		if ok && bare && i > 0 && (i > 1 || !isHelp(args[0])) {
			ok = false
		}
		switch {
		case !ok:
			inv.rest = append(inv.rest, arg)
		case c.name == "help":
			inv.help = true
		case inv.command == "" || inv.command == c.name:
			inv.command = c.name
		default:
			return inv, errUsage{fmt.Sprintf("%s and %s can't be combined", inv.command, c.name)}
		}
	}
	if inv.help && inv.command == "" && len(inv.rest) > 0 && !strings.HasPrefix(inv.rest[0], "-") {
		return inv, unknownCommand(inv.rest[0])
	}
	if inv.command != "" && !inv.help && len(inv.rest) > 0 {
		return inv, errUsage{fmt.Sprintf("%s takes no options, got %s", inv.command, strings.Join(inv.rest, " "))}
	}
	return inv, nil
}

func isHelp(token string) bool {
	c, ok := findCommand(token)
	return ok && c.name == "help"
}

// parseOptions reads the picker's options. Positional arguments left over are
// typos of a command, since the picker takes none.
func parseOptions(args []string) (options, error) {
	var o options
	fs := flag.NewFlagSet("gh select", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&o.noCache, "no-cache", false, "")
	fs.BoolVar(&o.noCache, "n", false, "")
	fs.StringVar(&o.cloneDir, "dir", "", "")
	fs.StringVar(&o.cloneDir, "d", "", "")
	fs.StringVar(&o.downloadDir, "download-dir", "", "")
	fs.BoolVar(&o.printPath, "print-path", false, "")
	fs.BoolVar(&o.printPath, "p", false, "")
	fs.StringVar(&o.theme, "theme", "", "")
	fs.StringVar(&o.border, "border", "", "")
	fs.StringVar(&o.icons, "icons", "", "")
	fs.BoolVar(&o.transparent, "transparent", false, "")
	if err := fs.Parse(args); err != nil {
		return o, optionError(err, fs)
	}
	if fs.NArg() > 0 {
		return o, unknownCommand(fs.Arg(0))
	}
	return o, nil
}

func optionError(err error, fs *flag.FlagSet) error {
	const undefined = "flag provided but not defined: "
	msg := err.Error()
	if !strings.HasPrefix(msg, undefined) {
		if name, ok := strings.CutPrefix(msg, "flag needs an argument: "); ok {
			return errUsage{dashed(strings.TrimLeft(name, "-")) + " needs a value"}
		}
		return errUsage{strings.Replace(msg, "flag ", "option ", 1)}
	}
	name := strings.TrimLeft(strings.TrimPrefix(msg, undefined), "-")
	var known []string
	fs.VisitAll(func(f *flag.Flag) {
		if len(f.Name) > 1 {
			known = append(known, "--"+f.Name)
		}
	})
	for _, c := range commands {
		known = append(known, "--"+c.name)
	}
	return errUsage{"unknown option " + dashed(name) + suggest(name, known)}
}

func unknownCommand(input string) error {
	names := make([]string, 0, len(commands))
	for _, c := range commands {
		names = append(names, c.name)
	}
	return errUsage{"unknown command " + input + suggest(input, names)}
}

func dashed(name string) string {
	if len(name) == 1 {
		return "-" + name
	}
	return "--" + name
}

func suggest(input string, candidates []string) string {
	cleaned := strings.TrimLeft(input, "-")
	best, bestDistance := "", len(cleaned)+1
	for _, c := range candidates {
		if d := levenshtein(cleaned, strings.TrimLeft(c, "-")); d < bestDistance {
			best, bestDistance = c, d
		}
	}
	if best == "" || bestDistance > max(2, len(cleaned)/2) {
		return ""
	}
	return ", did you mean " + best + "?"
}

func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// painter colors help output only for a terminal that hasn't set NO_COLOR.
type painter bool

func painterFor(f *os.File) painter {
	return painter(os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(f.Fd())))
}

func (p painter) paint(code, s string) string {
	if !p || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (p painter) bold(s string) string    { return p.paint("1", s) }
func (p painter) dim(s string) string     { return p.paint("2", s) }
func (p painter) red(s string) string     { return p.paint("31", s) }
func (p painter) green(s string) string   { return p.paint("32", s) }
func (p painter) magenta(s string) string { return p.paint("35", s) }
func (p painter) cyan(s string) string    { return p.paint("36", s) }

func printUsageError(err error) {
	p := painterFor(os.Stderr)
	msg := err.Error()
	if head, hint, ok := strings.Cut(msg, ", did you mean "); ok {
		msg = head + p.dim(", did you mean ") + p.cyan(strings.TrimSuffix(hint, "?")) + p.dim("?")
	}
	fmt.Fprintf(os.Stderr, "  %s %s\n  %s %s %s\n",
		p.red("error"), msg, p.dim("run"), p.cyan("gh select help"), p.dim("for usage"))
}

func printHelp(w *os.File) {
	p := painterFor(w)
	row := func(name, flags, desc string) string {
		return "    " + p.cyan(fmt.Sprintf("%-28s", name)) + p.magenta(fmt.Sprintf("%-10s", flags)) + p.dim(desc)
	}
	option := func(flags, desc string, more ...string) []string {
		lines := []string{"    " + p.magenta(fmt.Sprintf("%-22s", flags)) + desc}
		for _, m := range more {
			lines = append(lines, "    "+strings.Repeat(" ", 22)+p.dim(m))
		}
		return lines
	}
	env := func(name, desc string) string {
		return "    " + p.green(fmt.Sprintf("%-26s", name)) + p.dim(desc)
	}
	same := func(name, flag string) string {
		return "    " + p.green(fmt.Sprintf("%-26s", name)) + p.dim("same as ") + p.magenta(flag)
	}
	example := func(cmd, desc string) string {
		return "    " + p.cyan(fmt.Sprintf("%-34s", cmd)) + p.dim(desc)
	}

	lines := []string{
		"",
		"  " + p.bold(p.magenta("gh select")) + "  " + p.dim("find, preview and clone your GitHub repositories"),
		"",
		"  " + p.bold("Usage"),
		row("gh select [options]", "", "open the repo picker"),
		row("gh select <command>", "", "run one command and exit"),
		"",
		"  " + p.bold("Commands") + "  " + p.dim("each also takes its ") + p.magenta("-x") +
			p.dim(" and ") + p.magenta("--name") + p.dim(" form, and ") + p.magenta("-h"),
	}
	for _, c := range commands {
		flags := c.short
		if flags == "" {
			flags = "--" + c.name
		}
		lines = append(lines, row("gh select "+c.name+c.args, flags, c.summary))
	}
	lines = append(lines, "", "  "+p.bold("Options"))
	lines = append(lines, option("-n, --no-cache", "skip the cache and fetch fresh data")...)
	lines = append(lines, option("-d, --dir DIR", "clone into DIR instead of the current directory")...)
	lines = append(lines, option("--download-dir DIR", "save release assets and files into DIR")...)
	lines = append(lines, option("-p, --print-path", "print the picked repo's path, cloning it if needed")...)
	lines = append(lines, option("--theme NAME", "color theme, catppuccin by default",
		wrap(strings.Join(ui.ThemeNames(), ", ")+", or a custom ~/.config/gh-select/themes/NAME.json."+
			" ctrl+t in the app cycles them and saves the choice", 52)...)...)
	lines = append(lines, option("--border NAME", "panel border, none by default", strings.Join(ui.BorderNames(), ", "))...)
	lines = append(lines, option("--icons NAME", "nerd (default, needs a Nerd Font) or none")...)
	lines = append(lines, option("--transparent", "keep the terminal's background")...)
	lines = append(lines,
		"",
		"  "+p.bold("Environment")+"  "+p.dim("flags win over these"),
		same("GH_SELECT_CLONE_DIR", "--dir"),
		same("GH_SELECT_DOWNLOAD_DIR", "--download-dir"),
		same("GH_SELECT_THEME", "--theme"),
		same("GH_SELECT_BORDER", "--border"),
		same("GH_SELECT_ICONS", "--icons"),
		same("GH_SELECT_TRANSPARENT", "--transparent"),
		env("GH_SELECT_SCAN_DIRS", "folders searched first for clones, split like PATH"),
		env("GH_SELECT_CACHE_TTL", "seconds before the repo cache is stale, 1800"),
		env("NO_COLOR", "plain help and error output"),
		"",
		"  "+p.bold("In the app")+"  "+p.dim("press ")+p.cyan("?")+p.dim(" for every key"),
		"    "+p.dim("type to filter, enter acts on a repo, tab switches to starred and GitHub"),
		"    "+p.dim("in a repo's tree, space marks folders and c clones only those"),
		"    "+p.dim("ctrl+x marks repos you own and ctrl+d deletes them on GitHub, which"),
		"    "+p.dim("needs the delete_repo scope: gh auth refresh -s delete_repo"),
		"",
		"  "+p.bold("Examples"),
		example(`cd "$(gh select -p)"`, "jump into a repo, cloning it first"),
		example("gh select -d ~/code --theme nord", "clone under ~/code, in nord"),
		example("gh select limits", "see how much API budget is left"),
		example("gh select help refresh", "details on one command"),
		"",
	)
	fmt.Fprintln(w, strings.Join(lines, "\n"))
}

func printCommandHelp(w *os.File, name string) {
	c, _ := findCommand(name)
	p := painterFor(w)
	aliases := "--" + c.name
	if c.short != "" {
		aliases = c.short + ", " + aliases
	}
	lines := []string{
		"",
		"  " + p.bold(p.magenta("gh select "+c.name)) + "  " + p.dim(c.summary),
		"",
		"  " + p.bold("Usage"),
		"    " + p.cyan("gh select "+c.name+c.args) + "  " + p.dim("or ") + p.magenta(aliases),
		"",
	}
	for _, d := range c.details {
		lines = append(lines, "  "+d)
	}
	lines = append(lines, "")
	fmt.Fprintln(w, strings.Join(lines, "\n"))
}

func wrap(text string, width int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		if line != "" && len(line)+1+len(word) > width {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
