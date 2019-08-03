package kits

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"
)

const DEBUG = false

var UTCP8 = time.FixedZone("UTC+8", 8*60*60)

func Errorf(msg string, val ...interface{}) {
	fmt.Fprintf(os.Stdout, "%s [E] %v\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(msg, val...))
}

func Infof(msg string, val ...interface{}) {
	fmt.Fprintf(os.Stdout, "%s [I] %v\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(msg, val...))
}

func GetTpl(tpl *template.Template, parms interface{}) string {
	sbuf := new(strings.Builder)
	err := tpl.Execute(sbuf, parms)
	if err != nil {
		panic(err)
	}
	ret := sbuf.String()
	if DEBUG {
		Infof(ret)
	}
	return ret
}
