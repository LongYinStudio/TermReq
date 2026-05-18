package main

import (
	"fmt"
	"os"
)

func main() {
	app := NewApp(os.Stdin, os.Stdout)
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
