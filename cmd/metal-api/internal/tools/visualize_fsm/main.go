package main

import (
	"log"
	"os"

	"github.com/looplab/fsm"
	pfsm "github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm"
)

func main() {
	f := fsm.NewFSM("", pfsm.Events, nil)
	dot := fsm.Visualize(f)
	if err := os.WriteFile("fsm.dot", []byte(dot), 0600); err != nil {
		log.Fatal(err)
	}
}
