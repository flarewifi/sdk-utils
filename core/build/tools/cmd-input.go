package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func AskCmdInput(question string) (string, error) {
	fmt.Printf("\n%s:\n", question)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
}
