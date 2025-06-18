package testutils

import (
	"fmt"

	"github.com/goccy/go-json"
)

func PrintJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	if len(b) != 0 {
		fmt.Println(string(b))
	}
}
