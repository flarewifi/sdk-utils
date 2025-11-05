package sdkutils

import (
	"fmt"

	"github.com/goccy/go-json"
)

func PrettyPrint(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println("Error pretty printing:", err)
	}
	fmt.Println(string(b))
}
