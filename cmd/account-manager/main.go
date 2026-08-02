package main

import (
	"os"

	"github.com/seongmin221/ai-account-manager/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:]))
}
