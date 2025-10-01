package main

import (
	"os"
)

func Must(err error) {
	if err != nil {
		logger.Error("Operation failed", "error", err)
		os.Exit(1)
	}
}
