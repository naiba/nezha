package main

import (
	"fmt"

	"github.com/jinzhu/copier"
)

type Cat struct {
	age     int
	name    string
	friends []string
}

func main() {
	a := Cat{7, "Wilson", []string{"Tom", "Tabata", "Willie"}}
	b := Cat{7, "Wilson", []string{"Tom", "Tabata", "Willie"}}
	c := Cat{7, "Wilson", []string{"Tom", "Tabata", "Willie"}}
	wilson := []*Cat{&a, &b, &c}
	nikita := []Cat{}
	copier.Copy(&nikita, &wilson)

	nikita[0].friends = append(nikita[0].friends, "Syd")

	fmt.Println(wilson[0])
	fmt.Println(nikita[0])
}
