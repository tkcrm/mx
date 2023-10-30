package cfg

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/cristalhq/aconfig"
)

const cellSeparator = "|"

func (c *config) generateMarkdown(l *aconfig.Loader) {
	var table [][]string

	table = append(table, []string{
		"Name", "Required", "Secret", "Default value", "Usage", "Example",
	})

	sizes := make([]int, len(table[0]))

	var lineSize int
	for i, cell := range table[0] {
		sizes[i] = utf8.RuneCountInString(cell) + 2
	}

	configFields := getConfigFields(l)
	for _, f := range configFields {
		cell := []string{
			"`" + f.envName + "`",
			boolIcon(f.isRequired),
			boolIcon(f.isSecret),
			codeBlock(f.defaultValue),
			f.usage,
			codeBlock(f.example),
		}
		table = append(table, cell)

		lineSize = 0
		for i, item := range cell {
			if size := utf8.RuneCountInString(item); size+2 > sizes[i] {
				sizes[i] = size + 2
			}

			lineSize += sizes[i] // recalculate line size
		}
	}

	var out strings.Builder
	_, _ = out.WriteString("# Environments\n\n")
	for i, row := range table {
		_, _ = out.WriteString(cellSeparator)

		for j, cell := range row {
			size := utf8.RuneCountInString(" " + cell + " ")

			data := strings.Repeat(" ", sizes[j]-size)

			_, _ = out.WriteString(" " + cell + " ")
			_, _ = out.WriteString(data)

			if len(row)-1 != j {
				_, _ = out.WriteString(cellSeparator)
			}
		}

		if i == 0 {
			_, _ = out.WriteString(cellSeparator)
			_, _ = out.WriteRune('\n')

			_, _ = out.WriteString(cellSeparator)
			for j, item := range sizes {
				dashes := strings.Repeat("-", item)
				_, _ = out.WriteString(dashes)

				if len(sizes)-1 != j {
					_, _ = out.WriteString(cellSeparator)
				}
			}
		}

		_, _ = out.WriteString(cellSeparator)
		_, _ = out.WriteRune('\n')
	}

	_, _ = fmt.Fprintln(c.out, out.String())

	if c.filePath != "" {
		if err := os.WriteFile(c.filePath, []byte(out.String()), os.ModePerm); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func boolIcon(value bool) string {
	if value {
		return "✅"
	}

	return " "
}

func codeBlock(val string) string {
	if val == "" {
		return val
	}

	return "`" + val + "`"
}
