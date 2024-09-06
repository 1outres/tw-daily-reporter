package main

import (
	"fmt"
	"os"

	"github.com/1outres/tw-daily-reporter/cmd/twdr/app"
)

func main() {
	app := app.New()

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
