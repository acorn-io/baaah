package main

import (
	"os"

	"github.com/ibuildthecloud/baaah/pkg/deepcopy"
)

func main() {
	deepcopy.Deepcopy(os.Args[1:]...)
}
