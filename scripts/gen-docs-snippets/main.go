// gen-docs-snippets generates MDX snippet files for all sitectl commands.
// Run via: make docs-snippets from the sitectl-docs root.
// Output goes to snippets/commands/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	drupalcmd "github.com/libops/sitectl-drupal/cmd"
	islecmd "github.com/libops/sitectl-isle/cmd"
	sitectlcmd "github.com/libops/sitectl/cmd"
	"github.com/libops/sitectl/pkg/plugin"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	outputDir     = "snippets/commands"
	autoGenHeader = "{/* Auto-generated from source. Run `make docs-snippets` to update. */}\n\n"
)

type generator struct {
	displayPrefix string
	root          *cobra.Command
}

func main() {
	// Core sitectl
	core := &generator{
		displayPrefix: "sitectl",
		root:          sitectlcmd.RootCmd,
	}

	// Isle plugin
	isleSdk := plugin.NewSDK(plugin.Metadata{
		Name:        "isle",
		Description: "Islandora (ISLE) utilities and migration tools",
	})
	islecmd.RegisterCommands(isleSdk)
	isle := &generator{
		displayPrefix: "sitectl isle",
		root:          isleSdk.RootCmd,
	}

	// Drupal plugin
	drupalSdk := plugin.NewSDK(plugin.Metadata{
		Name:        "drupal",
		Description: "Drupal utilities for sitectl",
	})
	drupalcmd.RegisterCommands(drupalSdk)
	drupal := &generator{
		displayPrefix: "sitectl drupal",
		root:          drupalSdk.RootCmd,
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output dir: %v\n", err)
		os.Exit(1)
	}

	var total int
	for _, gen := range []*generator{core, isle, drupal} {
		gen.root.DisableAutoGenTag = true
		count := gen.run()
		total += count
	}
	fmt.Printf("generated %d snippets\n", total)
}

func (g *generator) run() int {
	var count int
	g.walkCommands(g.root, func(cmd *cobra.Command) {
		slug := g.commandSlug(cmd)
		path := filepath.Join(outputDir, slug+".mdx")
		if err := os.WriteFile(path, []byte(g.renderSnippet(cmd)), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Println(path)
		count++
	})
	return count
}

func (g *generator) walkCommands(cmd *cobra.Command, fn func(*cobra.Command)) {
	for _, sub := range cmd.Commands() {
		if g.skipCommand(sub) {
			continue
		}
		fn(sub)
		g.walkCommands(sub, fn)
	}
}

func (g *generator) skipCommand(cmd *cobra.Command) bool {
	if cmd.Hidden {
		return true
	}
	name := cmd.Name()
	if name == "help" || name == "completion" {
		return true
	}
	// Skip thin plugin-passthrough wrappers added by core sitectl discovery
	if cmd.DisableFlagParsing && strings.TrimSpace(cmd.Long) == "" && !cmd.HasAvailableSubCommands() {
		return true
	}
	return false
}

func (g *generator) commandSlug(cmd *cobra.Command) string {
	path := cmd.CommandPath()
	prefix := strings.ReplaceAll(g.displayPrefix, " ", "-")
	if strings.HasPrefix(path, g.displayPrefix+" ") {
		rel := path[len(g.displayPrefix)+1:]
		return strings.ToLower(prefix + "-" + strings.ReplaceAll(rel, " ", "-"))
	}
	return strings.ToLower(prefix)
}

func (g *generator) buildUseLine(cmd *cobra.Command) string {
	path := cmd.CommandPath()

	var fullPath string
	if path == g.displayPrefix || strings.HasPrefix(path, g.displayPrefix+" ") {
		fullPath = path
	} else {
		fullPath = g.displayPrefix + " " + path
	}

	// Append args from Use (everything after the command name)
	useParts := strings.Fields(cmd.Use)
	if len(useParts) > 1 {
		fullPath += " " + strings.Join(useParts[1:], " ")
	}

	// For group commands (no RunE), append <command>
	if !cmd.Runnable() && cmd.HasAvailableSubCommands() {
		fullPath += " <command>"
	}

	return fullPath
}

var (
	reSingleQuoted = regexp.MustCompile(`'([^'\n]+)'`)
	reAngleArg     = regexp.MustCompile("([^`]|^)(<[A-Za-z][A-Za-z0-9-]*>)")
	reFlagName     = regexp.MustCompile("([^`]|^)(--[A-Za-z][A-Za-z0-9-]*)")
)

func processDescription(s string) string {
	s = reSingleQuoted.ReplaceAllString(s, "`${1}`")
	s = reAngleArg.ReplaceAllString(s, "${1}`${2}`")
	s = reFlagName.ReplaceAllString(s, "${1}`${2}`")
	return s
}

func collectLocalFlags(cmd *cobra.Command) []*pflag.Flag {
	var flags []*pflag.Flag
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Hidden {
			flags = append(flags, f)
		}
	})
	return flags
}

func (g *generator) renderSnippet(cmd *cobra.Command) string {
	var b strings.Builder
	b.WriteString(autoGenHeader)

	// Long description, falling back to Short
	desc := strings.TrimSpace(cmd.Long)
	if desc == "" {
		desc = strings.TrimSpace(cmd.Short)
	}
	if desc != "" {
		b.WriteString(processDescription(desc))
		b.WriteString("\n\n")
	}

	// Usage code block
	b.WriteString("```bash\n")
	b.WriteString(g.buildUseLine(cmd))
	b.WriteString("\n```\n")

	// Aliases
	if len(cmd.Aliases) > 0 {
		b.WriteString("\n**Aliases:** `")
		b.WriteString(strings.Join(cmd.Aliases, "`, `"))
		b.WriteString("`\n")
	}

	// Flags table (skip for DisableFlagParsing commands — they accept arbitrary args)
	if !cmd.DisableFlagParsing {
		flags := collectLocalFlags(cmd)
		if len(flags) > 0 {
			b.WriteString("\n| Flag | Default | Description |\n")
			b.WriteString("|------|---------|-------------|\n")
			for _, f := range flags {
				flagStr := "--" + f.Name
				if f.Shorthand != "" {
					flagStr = "-" + f.Shorthand + ", " + flagStr
				}
				defVal := f.DefValue
				if defVal == "" {
					defVal = " "
				} else {
					defVal = "`" + defVal + "`"
				}
				fmt.Fprintf(&b, "| `%s` | %s | %s |\n", flagStr, defVal, processDescription(f.Usage))
			}
		}
	}

	return b.String()
}
