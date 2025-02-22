package main

import (
	"fmt"
	"os"

	"git.front.kjuulh.io/kjuulh/scaffold/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Printf("scaffold failed: %s\n", err.Error())
		os.Exit(1)
	}
}
