package main

import (
	"fmt"

	"github.com/ericpollmann/dotenvx"
)

func main() {
	for _, env := range dotenvx.Environ() {
		fmt.Println(env)
	}
}
