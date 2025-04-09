package uci

import (
	"fmt"
	"strconv"
)

type FlareStorageInfo struct {
	Storage   string
	Partition string
	Partnum   int
	Expanded  bool
}

func GetFlareStorage() (*FlareStorageInfo, bool) {
	info := FlareStorageInfo{}

	devs, ok := UciTree.Get("flare", "storage", "device")
	if ok && len(devs) > 0 {
		info.Storage = devs[0]
	}

	parts, ok := UciTree.Get("flare", "storage", "partition")
	if ok && len(parts) > 0 {
		info.Partition = parts[0]
	} else {
		return nil, false
	}

	nums, ok := UciTree.Get("flare", "storage", "partnum")
	if ok && len(nums) > 0 {
		num, err := strconv.Atoi(nums[0])
		if err != nil {
			info.Partnum = 0
		}
		info.Partnum = num
	} else {
		return nil, false
	}

	exps, ok := UciTree.Get("flare", "storage", "expanded")
	if ok && len(exps) > 0 {
		info.Expanded = exps[0] == "1"
	}

	return &info, true
}

func SetFlareStorageExpanded(expanded bool) error {
	val := "0"
	if expanded {
		val = "1"
	}

	ok := UciTree.Set("flare", "storage", "expanded", val)
	if !ok {
		return fmt.Errorf("failed to set storage expanded")
	}

	return nil
}
