package testutils

import (
	"encoding/json"
	"fmt"
)

func PrintJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	if len(b) != 0 {
		fmt.Println(string(b))
	}
}
