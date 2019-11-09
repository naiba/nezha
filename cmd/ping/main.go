package main

import (
	"log"
	"os/exec"
)

func main() {
	cmd := exec.Command("ping", "qiongbi.net", "-c5")
	output, err := cmd.Output()
	log.Println("output:", string(output))
	log.Println("err:", err)
}
