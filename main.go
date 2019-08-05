package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/obase/apigen/kits"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const METADIR = ".apigen"

var ipaths string
var parent string
var update bool
var help bool
var version bool
var md5sum bool

func main() {

	flag.BoolVar(&md5sum, "md5sum", false, "calculate and generate md5sum")
	flag.StringVar(&ipaths, "ipaths", "", "-IPATH directory, multiple values separate by comma ','")
	flag.StringVar(&parent, "parent", "", "parent directory")
	flag.BoolVar(&update, "update", false, "update or not")
	flag.BoolVar(&help, "help", false, "help information")
	flag.BoolVar(&version, "version", false, "metadir version")
	flag.Parse()

	if help {
		fmt.Fprintf(os.Stdout, "Usage: %v [-help] [-version] [-update] [-parent <dir>] [-ipaths <paths>]\n", filepath.Base(os.Args[0]))
		flag.CommandLine.SetOutput(os.Stdout)
		flag.PrintDefaults()
		return
	}

	if md5sum {
		root, _ := os.Getwd()
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() && !strings.HasPrefix(info.Name(), ".") && !strings.HasSuffix(info.Name(), ".md5sum") {
				genmd5sum(path)
			}
			return nil
		})
		return
	}
	exepath, err := exec.LookPath(os.Args[0])
	if err != nil {
		kits.Errorf("lookup exec path failed: %v", err)
		return
	}
	metadir := filepath.Join(filepath.Dir(exepath), METADIR)
	if update {
		updatemd(metadir)
		return
	}
	if !kits.IsDir(metadir) {
		kits.Errorf("missing metadir: %v", metadir)
		kits.Infof(`please "apigen -update" to create: %v`, metadir)
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

func genmd5sum(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	h := md5.New()
	io.Copy(bufio.NewWriter(h), bufio.NewReader(file))
	md5sum := hex.EncodeToString(h.Sum(nil))
	sumfile, err := os.OpenFile(path+".md5sum", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer sumfile.Close()
	sumfile.WriteString(md5sum)

	return nil
}

/*
更新meta目录:
- version
- protoc
- protoc-gen-api.exe
- github.com/obase/api/x.proto
*/
var PROXY_SERVER = kits.Getenv("PROXY_SERVER", "http://obase.github.io")

const PATTERN_RESOURCE = "/apigen/%s/%s"

var resources = []string{
	"protoc",
	"protoc-gen-api",
	"version",
	"github.com/obase/api/x.proto",
	"google/protobuf/descriptor.proto",
}

func updatemd(metadir string) {
	if !kits.IsDir(metadir) {
		if err := os.MkdirAll(metadir, os.ModePerm); err != nil {
			kits.Errorf("mkdir metadir failed: %v, %v", metadir, err)
			return
		}
	}
	for _, rsc := range resources {
		// windows需要添加扩展名
		if runtime.GOOS == "windows" && strings.HasPrefix(rsc, "protoc") {
			rsc = rsc + ".exe"
		}
		url := PROXY_SERVER + fmt.Sprintf(PATTERN_RESOURCE, runtime.GOOS, rsc)
		path := filepath.Join(metadir, rsc)
		if !checkmd5sum(url, path) {
			kits.Infof("download %s to %s, waiting......", url, path)
			download(url, path)
		}
	}
}

func checkmd5sum(url string, path string) bool {
	if !kits.IsExist(path) {
		return false
	}
	rsp, err := http.Get(url + ".md5sum")
	if err != nil {
		return false
	}
	defer rsp.Body.Close()

	if rsp.StatusCode < 200 || rsp.StatusCode >= 400 {
		return false
	}
	data, err := ioutil.ReadAll(bufio.NewReader(rsp.Body))
	if err != nil {
		return false
	}
	md5sum1 := strings.TrimSpace(string(data))

	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	h := md5.New()
	_, err = io.Copy(bufio.NewWriter(h), bufio.NewReader(file))
	if err != nil {
		return false
	}
	md5sum2 := hex.EncodeToString(h.Sum(nil))

	return md5sum1 == md5sum2
}

func download(url string, path string) {

	rsp, err := http.Get(url)
	if err != nil {
		kits.Errorf("http get error: %v, %v", url, err)
		return
	}
	defer rsp.Body.Close()

	if rsp.StatusCode >= 400 || rsp.StatusCode < 200 {
		kits.Errorf("http get error: %v, %v", url, rsp.StatusCode)
		return
	}

	dir := filepath.Dir(path)
	if !kits.IsExist(dir) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			kits.Errorf("mkdir all error: %v, %v", dir, err)
			return
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		kits.Errorf("open file error: %v, %v", path, err)
		return
	}
	defer file.Close()

	_, err = io.Copy(bufio.NewWriter(file), bufio.NewReader(rsp.Body))
	if err != nil {
		kits.Errorf("write file error: %v, %v", path, err)
		return
	}

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
