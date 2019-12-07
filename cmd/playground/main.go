package main

import (
	"log"
	"os/exec"
)

func main() {

}

func cmdExec() {
	cmd := exec.Command("ping", "qiongbi.net", "-c2")
	output, err := cmd.Output()
	log.Println("output:", string(output))
	log.Println("err:", err)

	cmd = exec.Command("ping", "qiongbi", "-c2")
	output, err = cmd.Output()
	log.Println("output:", string(output))
	log.Println("err:", err)
}
