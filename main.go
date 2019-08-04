package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/obase/apigen/kits"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const METADIR = ".apigen"
const SPACE  byte = ' ' //空白

var parent string
var update string
var help bool
var version bool

func main() {

	flag.StringVar(&parent, "parent", "", "parent directory")
	flag.StringVar(&update, "update", "", "update or not")
	flag.BoolVar(&help, "help", false, "help information")
	flag.BoolVar(&version, "version", false, "metadir version")
	flag.Parse()

	if help {
		fmt.Fprintf(os.Stdout, "Usage: %v [-help] [-version] [-parent <dir>] [-update <url>]\n", filepath.Base(os.Args[0]))
		flag.CommandLine.SetOutput(os.Stdout)
		flag.PrintDefaults()
		return
	}
	exepath, err := exec.LookPath(os.Args[0])
	if err != nil {
		kits.Errorf("lookup exec path failed: %v", err)
		return
	}
	metadir := filepath.Join(filepath.Dir(exepath), METADIR)
	if update != "" {
		updatemd(metadir, update)
		return
	}
	if !kits.IsDir(metadir) {
		kits.Infof(`Please execute "apigen -update <url>" to create/update metadir: %v`, metadir)
		return
	}
	if version {
		printversion(metadir, os.Stdout)
		return
	}

	if parent == "" {
		parent, _ = os.Getwd()
	}
	generate(metadir, parent)

}

/*
更新meta目录:
- version
- protoc
- protoc-gen-go
- github.com/obase/api/x.proto
- google/protobuf/descriptor.proto
*/
func updatemd(metadir string, server string) {

}

/*
创建proto文件
<metadir>/protoc --plugin=protoc-gen-go=<metadir>/proto-gen-go --go_out=plugins=grpc+apix:. --proto_path=<metadir> --proto_path=api xxx.proto yyy.proto
*/
func generate(metadir string, parent string) {
	apidir := filepath.Join(parent, "api")
	kits.Infof("path: %v, scanning......", apidir)
	if !kits.IsDir(apidir) {
		return
	}
	var protos []string
	filepath.Walk(apidir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".proto") {
			if relpath, err := filepath.Rel(apidir, path); err == nil {
				proto := strings.ReplaceAll(relpath, "\\", "/")
				protos = append(protos, proto)
				kits.Infof("file: %v", proto)
			}
		}
		return nil
	})

	if len(protos) > 0 {
		// <metadir>/protoc --plugin=protoc-gen-go=<metadir>/proto-gen-go --go_out=plugins=grpc+apix:<apidir> --proto_path=<metadir> --proto_path=<apidir> xxx.proto yyy.proto
		buf := new(bytes.Buffer)
		buf.WriteString(metadir)
		buf.WriteRune(os.PathSeparator)
		buf.WriteString("protoc")
		buf.WriteByte(SPACE);
		buf.WriteString("--plugin=protoc-gen-go=")
		buf.WriteString(metadir)
		buf.WriteRune(os.PathSeparator)
		buf.WriteString("protoc-gen-go")
		buf.WriteByte(SPACE)
		buf.WriteString("--go_out=plugins=grpc+apix:")
		buf.WriteString(apidir)
		buf.WriteByte(SPACE)
		buf.WriteString("--proto_path=")
		buf.WriteString(metadir)
		buf.WriteByte(SPACE)
		buf.WriteString("--proto_path=")
		buf.WriteString(apidir)
		buf.WriteByte(SPACE)
		for _, proto := range protos {
			buf.WriteString(proto)
			buf.WriteByte(SPACE)
		}
		fmt.Println(buf.String())
	}
}

/*
打印当前版本
*/
func printversion(metadir string, out io.Writer) {
	file, err := os.Open(filepath.Join(metadir, "version"))
	if err != nil {
		kits.Errorf("print version failed: %v", err)
		return
	}
	defer file.Close()
	io.Copy(out, file)
	fmt.Fprintln(out)
}
