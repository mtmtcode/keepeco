package cmd

import (
	"fmt"
	"os"
)

func abort(msg string, cause error) {
	err := fmt.Errorf(msg+": %w", cause)
	fmt.Printf("%+v\n", err)
	os.Exit(1)
}
