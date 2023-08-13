package cfg

import (
	"flag"
	"fmt"
	"strings"

	"github.com/cristalhq/aconfig"
)

func (c *config) generateDefaultEnvs(field aconfig.Field) bool {
	value := field.Tag("default")
	names := field.Tag("env")
	usage := field.Tag("usage")

	current := field
	if value == "" {
		value = "<empty>"
	}

	pad := 50

	var ok bool
	for {
		if current, ok = current.Parent(); !ok {
			break
		}

		names = fmt.Sprintf("%s_%s", current.Tag("env"), names)
	}

	var line strings.Builder
	_, _ = line.WriteString(names)
	_, _ = line.WriteString("=")
	_, _ = line.WriteString(value)

	if usage != "" {
		_, _ = line.WriteString(strings.Repeat(" ", pad-line.Len()))
		_, _ = line.WriteString("# " + usage)
	}

	c.print(line.String())

	return true
}

func (c *config) renderHelp(l *aconfig.Loader, fs *flag.FlagSet) {
	output := fs.Output()

	_, _ = fmt.Fprintln(output, "Usage:")
	_, _ = fmt.Fprintln(output)

	c.renderFlags(fs)

	var out strings.Builder

	_, _ = fmt.Fprintf(output, "\nDefault envs:\n%s\n", out.String())

	l.WalkFields(c.generateDefaultEnvs)
}
