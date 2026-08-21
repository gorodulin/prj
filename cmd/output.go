package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/format"
	"github.com/gorodulin/prj/internal/project"
)

// resolveOutputFormat returns the output format string for a project-list
// command. Priority: --json/--jsonl shorthands > explicit --format flag >
// list_format config > empty (format.DefaultFormat applies downstream).
func resolveOutputFormat(cmd *cobra.Command, cfg config.Config) string {
	formatStr, _ := cmd.Flags().GetString("format")
	if !cmd.Flags().Changed("format") && cfg.ListFormat != "" {
		formatStr = cfg.ListFormat
	}
	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		formatStr = "json"
	}
	if jsonlOut, _ := cmd.Flags().GetBool("jsonl"); jsonlOut {
		formatStr = "jsonl"
	}
	return formatStr
}

// writeProjectList renders projects to stdout in the given format.
func writeProjectList(projects []project.Project, formatStr string, cfg config.Config) error {
	opts := format.Options{
		Color: format.ResolveColor(os.Stdout, noColor, cfg.Color),
	}
	return format.Format(os.Stdout, projects, formatStr, opts)
}

// filterProjects returns projects that match any of the query terms (OR)
// and all of the required tags (AND). Terms may be any case; they are
// lower-cased here once.
func filterProjects(projects []project.Project, terms []string, tags []string) []project.Project {
	if len(terms) == 0 && len(tags) == 0 {
		return projects
	}
	lowered := make([]string, 0, len(terms))
	for _, term := range terms {
		lowered = append(lowered, strings.ToLower(term))
	}
	var result []project.Project
	for _, p := range projects {
		if matchesAnyQueryTerm(p, lowered) && matchesTags(p, tags) {
			result = append(result, p)
		}
	}
	return result
}

// matchesAnyQueryTerm reports whether any term is a substring of the
// project's ID, title, or any tag. Terms must already be lower-cased.
// An empty term list matches everything.
func matchesAnyQueryTerm(p project.Project, terms []string) bool {
	if len(terms) == 0 {
		return true
	}
	id := strings.ToLower(p.ID)
	title := strings.ToLower(p.Title)
	for _, q := range terms {
		if strings.Contains(id, q) || strings.Contains(title, q) {
			return true
		}
		for _, t := range p.Tags {
			if strings.Contains(strings.ToLower(t), q) {
				return true
			}
		}
	}
	return false
}

// matchesTags reports whether the project has every required tag (exact
// match, AND logic).
func matchesTags(p project.Project, tags []string) bool {
	for _, required := range tags {
		found := false
		for _, t := range p.Tags {
			if t == required {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
