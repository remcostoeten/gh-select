package main

import (
	"errors"
	"strings"
	"testing"
)

func TestRoute(t *testing.T) {
	cases := []struct {
		args    []string
		command string
		help    bool
		rest    []string
	}{
		{nil, "", false, nil},
		{[]string{"limits"}, "limits", false, nil},
		{[]string{"-l"}, "limits", false, nil},
		{[]string{"--limits"}, "limits", false, nil},
		{[]string{"--doctor"}, "doctor", false, nil},
		{[]string{"-v"}, "version", false, nil},
		{[]string{"help"}, "", true, nil},
		{[]string{"--help"}, "", true, nil},
		{[]string{"help", "refresh"}, "refresh", true, nil},
		{[]string{"refresh", "-h"}, "refresh", true, nil},
		{[]string{"-n", "--help"}, "", true, []string{"-n"}},
		{[]string{"-d", "refresh", "-p"}, "", false, []string{"-d", "refresh", "-p"}},
	}
	for _, c := range cases {
		inv, err := route(c.args)
		if err != nil {
			t.Errorf("route(%v) error: %v", c.args, err)
			continue
		}
		if inv.command != c.command || inv.help != c.help || strings.Join(inv.rest, " ") != strings.Join(c.rest, " ") {
			t.Errorf("route(%v) = %+v, want command %q help %v rest %v", c.args, inv, c.command, c.help, c.rest)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"limts"}, "unknown command limts, did you mean limits?"},
		{[]string{"help", "nope"}, "unknown command nope"},
		{[]string{"refresh", "-n"}, "refresh takes no options, got -n"},
		{[]string{"-r", "-l"}, "refresh and limits can't be combined"},
		{[]string{"--tehme", "nord"}, "unknown option --tehme, did you mean --theme?"},
		{[]string{"--dir"}, "--dir needs a value"},
		{[]string{"--theme", "nord", "-r"}, "refresh takes no options, got --theme nord"},
		{[]string{"-n", "stray"}, "unknown command stray"},
	}
	for _, c := range cases {
		inv, err := route(c.args)
		if err == nil {
			_, err = parseOptions(inv.rest)
		}
		var usage errUsage
		if !errors.As(err, &usage) || usage.msg != c.want {
			t.Errorf("%v: got %v, want %q", c.args, err, c.want)
		}
	}
}

func TestParseOptions(t *testing.T) {
	o, err := parseOptions([]string{"-n", "-d", "~/code", "--theme", "nord", "-p", "--transparent"})
	if err != nil {
		t.Fatal(err)
	}
	if !o.noCache || o.cloneDir != "~/code" || o.theme != "nord" || !o.printPath || !o.transparent {
		t.Fatalf("options = %+v", o)
	}
}
