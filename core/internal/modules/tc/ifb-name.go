package tc

import "fmt"

func ifbName(dev string) string {
	return fmt.Sprintf("%s-ifb", dev)
}
