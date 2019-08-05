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
	"runtime"
	"strings"
)

const METADIR = ".apigen"

var ipaths string
var parent string
var update string
var help bool
var version bool

func main() {

	flag.StringVar(&ipaths, "ipaths", "", "-IPATH directory, multiple values separate by comma ','")
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
	generate(metadir, parent, ipaths)

}

/*
更新meta目录:
- version
- protoc
- protoc-gen-api.exe
- github.com/obase/api/x.proto
*/
func updatemd(metadir string, server string) {

}

/*
创建proto文件
<metadir>/protoc --plugin=protoc-gen-go=<metadir>/proto-gen-go --go_out=plugins=grpc+apix:. --proto_path=<metadir> --proto_path=api xxx.proto yyy.proto
*/
func generate(metadir string, parent string, ipaths string) {
	apidir := filepath.Join(parent, "api")
	kits.Infof("path: %v, scanning......", apidir)
	if !kits.IsDir(apidir) {
		return
	}
	// 生成命令行及参数
	cmdname, cmdargs, protoidx := command(metadir, apidir, ipaths)
	filepath.Walk(apidir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".proto") {
			if relpath, err := filepath.Rel(apidir, path); err == nil {
				// 1. 删除旧的go文件
				gofile := path[:len(path)-6] + ".pb.go"
				if kits.IsExist(gofile) {
					_ = os.Remove(gofile)
				}
				// 2. 创建新的go文件
				proto := strings.ReplaceAll(relpath, "\\", "/")
				kits.Infof("file: %v, generating......", proto)
				cmdargs[protoidx] = proto
				cmd := exec.Command(cmdname, cmdargs...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					kits.Errorf("generate failed: %v, err=%v", proto, err)
				}
			}
		}
		return nil
	})

}

// <metadir>/protoc --plugin=protoc-gen-go=<metadir>/proto-gen-go --go_out=plugins=grpc+apix:<apidir> --proto_path=<metadir> --proto_path=<apidir> xxx.proto yyy.proto
func command(metadir string, apidir string, ipaths string) (cmd string, args []string, last int) {
	args = make([]string, 0, 5)

	// 一次性分配
	buf := bytes.NewBuffer(make([]byte, 256))

	buf.Reset()
	buf.WriteString(metadir)
	buf.WriteRune(os.PathSeparator)
	buf.WriteString("protoc")
	cmd = buf.String()

	buf.Reset()
	buf.WriteString("--plugin=protoc-gen-api=")
	buf.WriteString(metadir)
	buf.WriteRune(os.PathSeparator)
	buf.WriteString("protoc-gen-api")
	if runtime.GOOS == "windows" {
		buf.WriteString(".exe")
	}
	args = append(args, buf.String())

	buf.Reset()
	buf.WriteString("--api_out=plugins=grpc+apix:")
	buf.WriteString(apidir)
	args = append(args, buf.String())

	if ipaths != "" {
		for _, ipath := range strings.Split(ipaths, ",") {
			buf.Reset()
			buf.WriteString("--proto_path=")
			buf.WriteString(ipath)
			args = append(args, buf.String())
		}
	}

	buf.Reset()
	buf.WriteString("--proto_path=")
	buf.WriteString(metadir)
	args = append(args, buf.String())

	buf.Reset()
	buf.WriteString("--proto_path=")
	buf.WriteString(apidir)
	args = append(args, buf.String())
	last = len(args)
	// 扩展最后一个元素，否则会抛下标越界错误
	args = append(args, "")
	return
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
