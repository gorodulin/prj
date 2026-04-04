package cmd

import (
	"fmt"
	"strings"

	"github.com/gorodulin/prj/internal/config"
)

// requireConfig checks that the named config fields are all non-empty and
// returns a single error listing ALL missing fields with remediation hints.
func requireConfig(cfg config.Config, keys ...string) error {
	var missing []config.Field
	for _, key := range keys {
		f, ok := config.FieldByKey(key)
		if !ok {
			return fmt.Errorf("internal error: unknown config key %q", key)
		}
		if f.IsEmpty(&cfg) {
			missing = append(missing, f)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("missing required configuration:")
	for _, f := range missing {
		fmt.Fprintf(&b, "\n  prj config set %s %s", f.Key, f.Hint)
	}
	return fmt.Errorf("%s", b.String())
}
