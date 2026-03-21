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

	archivesspacecmd "github.com/libops/sitectl-archivesspace/cmd"
	drupalcmd "github.com/libops/sitectl-drupal/cmd"
	islecmd "github.com/libops/sitectl-isle/cmd"
	ojscmd "github.com/libops/sitectl-ojs/cmd"
	omekaclassiccmd "github.com/libops/sitectl-omeka-classic/cmd"
	omekascmd "github.com/libops/sitectl-omeka-s/cmd"
	wpcmd "github.com/libops/sitectl-wp/cmd"
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

// pluginGen builds a generator for a plugin by creating an SDK with the given
// metadata and running the plugin's RegisterCommands against it. The register
// callback wraps each plugin's entrypoint so signature differences (some return
// an error, most do not) are normalized here.
func pluginGen(name, description string, register func(*plugin.SDK) error) *generator {
	sdk := plugin.NewSDK(plugin.Metadata{
		Name:        name,
		Description: description,
	})
	if err := register(sdk); err != nil {
		fmt.Fprintf(os.Stderr, "register %s commands: %v\n", name, err)
		os.Exit(1)
	}
	return &generator{
		displayPrefix: "sitectl " + name,
		root:          sdk.RootCmd,
	}
}

func main() {
	// Output paths are written relative to the working directory. The module
	// lives at <docs-root>/scripts/gen-docs-snippets, so `make docs-snippets`
	// passes the docs root as the first argument and we chdir into it before
	// writing. This keeps the generator a self-contained module (its own
	// go.mod + replace directives) with no go.work file required.
	if len(os.Args) > 1 {
		if err := os.Chdir(os.Args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "chdir %s: %v\n", os.Args[1], err)
			os.Exit(1)
		}
	}

	// Core sitectl
	core := &generator{
		displayPrefix: "sitectl",
		root:          sitectlcmd.RootCmd,
	}

	// Plugins. Keep this list in sync with the repos in scripts/use-go-work.sh
	// and the require/replace blocks in scripts/gen-docs-snippets/go.mod.
	plugins := []*generator{
		pluginGen("isle", "Islandora (ISLE) utilities and migration tools", func(s *plugin.SDK) error {
			islecmd.RegisterCommands(s)
			return nil
		}),
		pluginGen("drupal", "Drupal utilities for sitectl", func(s *plugin.SDK) error {
			return drupalcmd.RegisterCommands(s)
		}),
		pluginGen("archivesspace", "ArchivesSpace helpers", func(s *plugin.SDK) error {
			archivesspacecmd.RegisterCommands(s)
			return nil
		}),
		pluginGen("ojs", "Open Journal Systems helpers", func(s *plugin.SDK) error {
			ojscmd.RegisterCommands(s)
			return nil
		}),
		pluginGen("omeka-classic", "Omeka Classic helpers", func(s *plugin.SDK) error {
			omekaclassiccmd.RegisterCommands(s)
			return nil
		}),
		pluginGen("omeka-s", "Omeka S helpers", func(s *plugin.SDK) error {
			omekascmd.RegisterCommands(s)
			return nil
		}),
		pluginGen("wp", "WordPress helpers", func(s *plugin.SDK) error {
			wpcmd.RegisterCommands(s)
			return nil
		}),
		// NOTE: sitectl-libops is intentionally not wired in here. The local
		// checkout imports github.com/libops/sitectl/pkg/format, which the
		// pinned/local sitectl module does not provide, so it cannot compile
		// against this workspace. Its docs under plugins/libops/ remain
		// hand-maintained until the two modules are realigned.
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output dir: %v\n", err)
		os.Exit(1)
	}

	var total int
	for _, gen := range append([]*generator{core}, plugins...) {
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
		if !f.Hidden && !strings.Contains(strings.ToLower(f.Usage), "deprecated alias") {
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
