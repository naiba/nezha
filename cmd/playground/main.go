package main

import (
	"log"

	"github.com/shirou/gopsutil/v3/host"
)

func main() {
	info, err := host.Info()
	if err != nil {
		panic(err)
	}
	log.Printf("%#v", info)
}
