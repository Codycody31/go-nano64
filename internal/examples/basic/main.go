package main

import (
	"fmt"

	"go.codycody31.dev/nano64"
)

func main() {
	id, err := nano64.GenerateDefault()
	if err != nil {
		panic(err)
	}

	fmt.Println(id.ToHex()) // 17‑char uppercase hex TIMESTAMP-RANDOM
	// 199C01B6659-5861C
	fmt.Println(id.ToBytes()) // [8]byte
	// [25 156 1 182 101 149 134 28]
	fmt.Println(id.GetTimestamp()) // ms since epoch
	// 1759864645209
}
