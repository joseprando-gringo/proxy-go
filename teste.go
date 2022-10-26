package main

import (
	"fmt"
	"strings"
)

func main1() {
	m := make(map[string][]string)
	a := make([]string, 1)
	a[0] = "teste"

	m["X-Forwarded-For"] = a
	if value, ok := m["X-Forwarded-For"]; ok {
		fmt.Println(strings.Join(value, ","))
	}
}
