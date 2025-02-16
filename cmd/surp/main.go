package main

import (
	"os"

	"github.com/burgrp/surp-go/cmd/surp/commands"
)

func main() {

	err := commands.GetRootCommand().Execute()
	if err != nil {
		os.Exit(1)
	}
}
