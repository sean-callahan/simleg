package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/sean-callahan/simleg"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s path", os.Args[0])
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	p := &simleg.Parser{}
	p.Use(f)

	var prog simleg.Program
	for {
		as, err := p.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalln("parse:", err)
		}
		prog = append(prog, as)
	}

	cpu := &simleg.CPU{}
	if err := cpu.Load(prog); err != nil {
		log.Fatalln("load program:", err)
	}

	for cpu.Step() {
	}
}
