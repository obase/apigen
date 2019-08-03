package main

import (
	"flag"
	"os"
	"path/filepath"
)

var parent string
var upgrade bool

func main() {

	flag.StringVar(&parent, "parent", "", "package directory")
	flag.BoolVar(&upgrade, "upgrade", false, "upgrade or not")

	if upgrade {
		doupgrade()
	} else {
		if parent == "" {
			parent = filepath.Base(os.Args[0])
		}
		doapigen(parent)
	}
	os.Exit(0)
}

func doupgrade() {

}

func doapigen(path string) {

}
