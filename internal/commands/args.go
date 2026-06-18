package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// This file provides drop-in replacements for cobra's positional-argument
// validators (ExactArgs, MinimumNArgs, MaximumNArgs, RangeArgs). Cobra's
// defaults emit terse, unhelpful errors such as "accepts 1 arg(s), received 0".
// The replacements below name the missing argument(s), show the usage line, and
// echo the command's own examples so the user can immediately self-correct.

// exactArgs enforces exactly n positional arguments.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		switch {
		case len(args) < n:
			return argErr(cmd, missingMsg(cmd, len(args), n))
		case len(args) > n:
			return argErr(cmd, fmt.Sprintf("too many arguments: expected %d, got %d", n, len(args)))
		}
		return nil
	}
}

// minArgs enforces at least n positional arguments.
//
//nolint:unused // part of the drop-in args-validator toolkit; retained for callers
func minArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < n {
			return argErr(cmd, missingMsg(cmd, len(args), n))
		}
		return nil
	}
}

// maxArgs enforces at most n positional arguments.
func maxArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > n {
			return argErr(cmd, fmt.Sprintf("too many arguments: expected at most %d, got %d", n, len(args)))
		}
		return nil
	}
}

// rangeArgs enforces between min and max positional arguments (inclusive).
//
//nolint:unused // part of the drop-in args-validator toolkit; retained for callers
func rangeArgs(min, max int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		switch {
		case len(args) < min:
			return argErr(cmd, missingMsg(cmd, len(args), min))
		case len(args) > max:
			return argErr(cmd, fmt.Sprintf("too many arguments: expected at most %d, got %d", max, len(args)))
		}
		return nil
	}
}

// missingMsg builds a "missing required argument" message, naming the specific
// placeholders (e.g. <name>, <old-name>) that were not supplied. It derives the
// names from the command's Use line and falls back to a count-based message when
// no placeholders are declared.
func missingMsg(cmd *cobra.Command, have, want int) string {
	names := argPlaceholders(cmd)
	if have < len(names) {
		missing := names[have:]
		if want < len(names) {
			missing = names[have:want]
		}
		if len(missing) == 1 {
			return fmt.Sprintf("missing required argument: %s", missing[0])
		}
		if len(missing) > 1 {
			return fmt.Sprintf("missing required arguments: %s", strings.Join(missing, ", "))
		}
	}
	return fmt.Sprintf("expected %d argument(s), got %d", want, have)
}

// argPlaceholders extracts the positional-argument tokens (e.g. <name>,
// [optional]) from a command's Use line, skipping the command verb itself.
// argPlaceholders returns the required positional placeholders (<...>) from a
// command's Use line. Optional tokens ([...]) are deliberately excluded: this
// list only names the missing *required* arguments, so folding optionals in
// would mislabel the error text.
func argPlaceholders(cmd *cobra.Command) []string {
	fields := strings.Fields(cmd.Use)
	if len(fields) <= 1 {
		return nil
	}
	var names []string
	for _, f := range fields[1:] {
		if strings.HasPrefix(f, "<") {
			names = append(names, f)
		}
	}
	return names
}

// EnforceSubcommandErrors walks the command tree and makes every command *group*
// (a command that only organizes subcommands and has no action of its own) reject
// an unknown subcommand with an actionable error, instead of cobra's default of
// silently printing the group's help with a success (exit 0) status. Silently
// printing help on a bad token hides the mistake and — because the exit code is
// 0 — lets scripts and CI treat a typo as success; every mainstream CLI (git,
// kubectl, docker, cargo) instead errors to stderr with a non-zero exit.
//
// The error adapts to the group:
//   - a group with a single subcommand points the user straight at that command
//     (with its example), since there is no ambiguity about what they meant;
//   - a group with several subcommands lists the valid subcommands so the user
//     can choose, and points at --help for full usage and examples.
//
// Running a group with no arguments still prints its help and exits 0.
func EnforceSubcommandErrors(root *cobra.Command) {
	for _, c := range root.Commands() {
		EnforceSubcommandErrors(c)
	}
	// Only groups: skip leaf commands and any command that does real work itself.
	if !root.HasSubCommands() || root.Runnable() {
		return
	}
	// Enable "Did you mean" suggestions (cobra leaves the threshold at 0 on
	// non-root commands, which disables Levenshtein matching).
	if root.SuggestionsMinimumDistance <= 0 {
		root.SuggestionsMinimumDistance = 2
	}
	root.Args = rejectUnknownSubcommand
	root.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return rejectUnknownSubcommand(cmd, args)
		}
		return cmd.Help()
	}
}

// HideFlagEverywhere walks the command tree and marks the named local flag hidden
// on every command that declares it. Used to keep auto-resolved plumbing flags
// (e.g. --cloud-provider) out of help output while leaving them usable as an
// explicit override.
func HideFlagEverywhere(root *cobra.Command, name string) {
	for _, c := range root.Commands() {
		HideFlagEverywhere(c, name)
	}
	if f := root.Flags().Lookup(name); f != nil {
		f.Hidden = true
	}
}

// rejectUnknownSubcommand is the Args validator installed on group commands by
// EnforceSubcommandErrors.
func rejectUnknownSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}

	var subs []*cobra.Command
	for _, c := range cmd.Commands() {
		if c.IsAvailableCommand() {
			subs = append(subs, c)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "unknown subcommand %q for %q", args[0], cmd.CommandPath())

	if sugg := cmd.SuggestionsFor(args[0]); len(sugg) > 0 {
		b.WriteString("\n\nDid you mean this?")
		for _, s := range sugg {
			fmt.Fprintf(&b, "\n\t%s", s)
		}
	}

	if len(subs) == 1 {
		only := subs[0]
		fmt.Fprintf(&b, "\n\nRun this instead:\n  %s %s", cmd.CommandPath(), only.Name())
		if only.Short != "" {
			fmt.Fprintf(&b, "    %s", only.Short)
		}
		if ex := strings.TrimRight(only.Example, "\n"); ex != "" {
			fmt.Fprintf(&b, "\n\nExample:\n%s", ex)
		}
	} else {
		b.WriteString("\n\nAvailable commands:")
		for _, s := range subs {
			fmt.Fprintf(&b, "\n  %-15s %s", s.Name(), s.Short)
		}
		fmt.Fprintf(&b, "\n\nRun '%s --help' for usage and examples.", cmd.CommandPath())
	}
	return fmt.Errorf("%s", b.String())
}

// argErr assembles the final, actionable error: the headline detail, the usage
// line, and the command's examples (if any).
func argErr(cmd *cobra.Command, detail string) error {
	var b strings.Builder
	b.WriteString(detail)
	fmt.Fprintf(&b, "\n\nUsage:\n  %s", cmd.UseLine())
	if ex := strings.TrimRight(cmd.Example, "\n"); ex != "" {
		fmt.Fprintf(&b, "\n\nExamples:\n%s", ex)
	}
	return fmt.Errorf("%s", b.String())
}
