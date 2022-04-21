package main

import (
	"os"

	"github.com/acorn-io/baaah/pkg/deepcopy"
)

func main() {
	deepcopy.Deepcopy(os.Args[1:]...)
}
