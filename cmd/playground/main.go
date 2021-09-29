package main

import (
	"log"

	"github.com/robfig/cron/v3"
)

func main() {
	c := cron.New(cron.WithSeconds())
	_, err := c.AddFunc("* * * * * *", func() {
		log.Println("bingo second")
	})
	if err != nil {
		panic(err)
	}
	_, err = c.AddFunc("* * * * *", func() {
		log.Println("bingo minute")
	})
	if err != nil {
		panic(err)
	}
	c.Start()
	select {}
}
