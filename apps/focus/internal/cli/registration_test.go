package cli

import (
	"fmt"
	"sort"
	"testing"

	"github.com/spf13/cobra"
)

// collectNames walks cmd's registered subcommands (skipping cobra's
// auto-injected help/completion) and appends their full paths to out.
func collectNames(cmd *cobra.Command, prefix string, out *[]string) {
	for _, sub := range cmd.Commands() {
		name := sub.Name()
		if name == "help" || name == "completion" {
			continue
		}
		*out = append(*out, prefix+name)
		collectNames(sub, prefix+name+" ", out)
	}
}

// TestRegisteredCommandSet pins the registered cobra command tree to the
// tool-name contract (analysis on card t_9e13b7f2): 48 unique commands
// besides the root, no name registered twice, dashboard and report absent.
func TestRegisteredCommandSet(t *testing.T) {
	var got []string
	collectNames(rootCmd, "", &got)

	if len(got) != 48 {
		t.Fatalf("registered command count = %d, want 48:\n%v", len(got), got)
	}

	seen := make(map[string]bool)
	for _, name := range got {
		if seen[name] {
			t.Errorf("command %q is registered more than once", name)
		}
		seen[name] = true
	}

	for _, must := range []string{
		"add", "remove", "template delete", "bulk delete", "config reset",
		"journal view", "journal search", "calendar", "list",
	} {
		if !seen[must] {
			t.Errorf("command %q missing from the registered tree", must)
		}
	}

	for _, banned := range []string{"dashboard", "report"} {
		if seen[banned] {
			t.Errorf("command %q must not be registered", banned)
		}
	}

	sorted := append([]string(nil), got...)
	sort.Strings(sorted)
	fmt.Println("registered:", sorted)
}
