package nftables

import (
	"bytes"
	"fmt"

	"github.com/goccy/go-json"

	"core/internal/utils/cmd"
)

type NftListMapResult struct {
	Nftables []*NftablesData `json:"nftables"`
}

type NftablesData struct {
	Map *MapData `json:"map"`
}

type MapData struct {
	Elem [][]*MapElem `json:"elem"`
}

type MapElem struct {
	Elem *MapElemVal `json:"elem"`
}

type MapElemVal struct {
	Val     string       `json:"val"`
	Counter *ElemCounter `json:"counter"`
}

type ElemCounter struct {
	Packets uint `json:"packets"`
	Bytes   uint `json:"bytes"`
}

type StatData struct {
	Bytes   uint
	Packets uint
}

type StatResult struct {
	MacStats map[string]StatData
	IpStats  map[string]StatData
}

func GetStats() (stat StatResult, err error) {
	result, err := nftQue.Exec(func() (interface{}, error) {
		nftlistmac, err := nftListMap(connMacMap)
		if err != nil {
			return nil, err
		}

		nftlistip, err := nftListMap(connIpMap)
		if err != nil {
			return nil, err
		}

		macstat := resultMap(nftlistmac)
		ipstat := resultMap(nftlistip)

		result := StatResult{
			MacStats: macstat,
			IpStats:  ipstat,
		}

		return result, nil
	})

	if err != nil {
		return StatResult{}, err
	}

	return result.(StatResult), nil
}

func nftListMap(mapname string) (*NftListMapResult, error) {
	var out bytes.Buffer

	command := fmt.Sprintf("nft -n -j list map ip internet %s", mapname)
	if err := cmd.ExecOutput(command, &out); err != nil {
		return nil, err
	}

	var result NftListMapResult
	err := json.Unmarshal(out.Bytes(), &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func resultMap(data *NftListMapResult) map[string]StatData {
	stats := map[string]StatData{}

	for _, d := range data.Nftables {
		if d.Map != nil {
			m := d.Map
			if m.Elem != nil {
				for _, elems := range m.Elem {
					for _, elem := range elems {
						if elem.Elem != nil {
							stat := StatData{
								Packets: elem.Elem.Counter.Packets,
								Bytes:   elem.Elem.Counter.Bytes,
							}
							stats[elem.Elem.Val] = stat
						}
					}
				}
			}
		}
	}

	return stats
}
