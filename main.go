package main

import (
	"fmt"

	"github.com/snuskin-1313/gator/internal/config"
)

func main() {
	c := config.Read()
	c.SetUser("ryuko matoi")
	c = config.Read()
	fmt.Println(c)
}
