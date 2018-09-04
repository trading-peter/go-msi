// +build windows

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func main() {
	confirm(rmFile("log-install.txt"), "install log removal")
	confirm(rmFile("log-uninstall.txt"), "uninstall log removal")

	mustNotBeInstalled()

	gopath := os.Getenv("GOPATH")
	wd := makeDir(filepath.Join(gopath, "src/github.com/mat007/go-msi/testing/hello"))
	mustContains(wd, "hello.go")
	mustChdir(wd)

	helloBuild := makeCmd("go", "build", "-o", "build/amd64/hello.exe", "hello.go")
	mustExec(helloBuild, "hello build failed %v")

	setup := makeCmd("C:/go-msi/go-msi.exe", "set-guid")
	mustExec(setup, "Packaging setup failed %v")

	msi := "hello.msi"
	version := "v12.34.5678"
	pkg := makeCmd("C:/go-msi/go-msi.exe", "make",
		"--msi", msi,
		"--version", version,
		"--arch", "amd64",
		"--property", "SOME_VERSION=some version",
		"--keep",
	)
	mustExec(pkg, "Packaging failed %v")
	mustExist(msi, "Package file is missing %v")

	packageInstall := makeCmd("msiexec", "/i", msi, "/q", "/log", "log-install.txt", "MINIMUMBUILD=0")
	mustFail(packageInstall.Exec(), "Package installation succeeded")
	content := readLog("log-install.txt")
	mustContain(content, "Product: hello -- some condition message")
	mustContain(content, "Product: hello -- Installation failed")
	mustSucceed(rmFile("log-install.txt"), "rmfile failed %v")

	packageInstall = makeCmd("msiexec", "/i", msi, "/q", "/log", "log-install.txt")
	mustExec(packageInstall, "Package installation failed %v")
	content = readLog("log-install.txt")
	mustContain(content, "Product: hello -- Installation completed successfully.")
	mustContain(content, "Windows Installer installed the product. Product Name: hello. Product Version: 12.34.5678. Product Language: 1033. Manufacturer: mh-cbon.")
	mustSucceed(rmFile("log-install.txt"), "rmfile failed %v")

	mustBeInstalled(version)

	// Re-installing the same package in quiet mode re-configures it… to the same state.
	packageReinstall := makeCmd("msiexec", "/i", msi, "/q", "/log", "log-install.txt")
	mustExec(packageReinstall, "Package re-installation failed %v")
	content = readLog("log-install.txt")
	mustContain(content, "Product: hello -- Configuration completed successfully.")
	mustContain(content, "Windows Installer reconfigured the product. Product Name: hello. Product Version: 12.34.5678. Product Language: 1033. Manufacturer: mh-cbon.")
	mustSucceed(rmFile("log-install.txt"), "rmfile failed %v")

	mustBeInstalled(version)

	msi2 := "hello-2.msi"
	pkg = makeCmd("C:/go-msi/go-msi.exe", "make",
		"--msi", msi2,
		"--version", "v12.34.5677", // version down
		"--arch", "amd64",
		"--property", "SOME_VERSION=some version",
		"--keep",
	)
	mustExec(pkg, "Packaging failed %v")
	mustExist(msi2, "Package file is missing %v")
	packageDowngrade := makeCmd("msiexec", "/i", msi2, "/q", "/log", "log-install.txt")
	mustFail(packageDowngrade.Exec(), "Package downgrade succeeded")
	content = readLog("log-install.txt")
	mustContain(content, "Product: hello -- A newer version of this software is already installed.")
	mustSucceed(rmFile("log-install.txt"), "rmfile failed %v")

	mustBeInstalled(version)

	version = "v12.34.5679" // version up
	pkg = makeCmd("C:/go-msi/go-msi.exe", "make",
		"--msi", msi2,
		"--version", version,
		"--arch", "amd64",
		"--property", "SOME_VERSION=some version",
		"--keep",
	)
	mustExec(pkg, "Packaging failed %v")
	mustExist(msi2, "Package file is missing %v")
	packageUpgrade := makeCmd("msiexec", "/i", msi2, "/q", "/log", "log-install.txt")
	mustExec(packageUpgrade, "Package upgrade failed %v")
	content = readLog("log-install.txt")
	mustContain(content, "Product: hello -- Installation completed successfully.")
	mustContain(content, "Windows Installer installed the product. Product Name: hello. Product Version: 12.34.5679. Product Language: 1033. Manufacturer: mh-cbon.")
	mustSucceed(rmFile("log-install.txt"), "rmfile failed %v")

	mustBeInstalled(version)

	packageUninstall := makeCmd("msiexec", "/x", msi, "/q", "/log", "log-uninstall.txt")
	mustFail(packageUninstall.Exec(), "Old package uninstall succeeded")
	content = readLog("log-uninstall.txt")
	mustContain(content, "This action is only valid for products that are currently installed")
	mustSucceed(rmFile("log-uninstall.txt"), "rmfile failed %v")

	mustBeInstalled(version)

	packageUninstall = makeCmd("msiexec", "/x", msi2, "/q", "/log", "log-uninstall.txt")
	mustExec(packageUninstall, "Package uninstall failed %v")
	content = readLog("log-uninstall.txt")
	mustContain(content, "Product: hello -- Removal completed successfully.")
	mustContain(content, "Windows Installer removed the product. Product Name: hello. Product Version: 12.34.5679. Product Language: 1033. Manufacturer: mh-cbon.")
	mustSucceed(rmFile("log-uninstall.txt"), "rmfile failed %v")

	mustNotBeInstalled()

	version = "0.0.1"
	msi = "hello-choco.msi"
	pkg = makeCmd("C:/go-msi/go-msi.exe", "make",
		"--msi", msi,
		"--version", version,
		"--arch", "amd64",
		"--property", "SOME_VERSION=some version",
		"--keep",
	)
	mustExec(pkg, "Packaging failed %v")
	mustExist(msi, "Package file is missing %v")
	chocoPkg := makeCmd("C:/go-msi/go-msi.exe", "choco",
		"--input", msi,
		"--version", version,
		"-c", `"C:\Program Files\changelog\changelog.exe" ghrelease --version `+version,
		"--keep",
	)
	mustExec(chocoPkg, "hello choco package make failed %v")
	mustExist("hello."+version+".nupkg", "Chocolatey nupkg file is missing %v")

	chocoInstall := makeCmd("choco", "install", "hello."+version+".nupkg", "-y")
	mustExec(chocoInstall, "hello choco package install failed %v")

	mustBeInstalled("v" + version)

	chocoUninstall := makeCmd("choco", "uninstall", "hello", "-v", "-d", "-y", "--force")
	mustExec(chocoUninstall, "hello choco package uninstall failed %v")
	readFile(`C:\ProgramData\chocolatey\logs\chocolatey.log`)

	mustNotBeInstalled()

	fmt.Println("\nSuccess!")
}

const (
	url     = "http://localhost:8080/"
	service = "HelloSvc"
)

func mustBeInstalled(version string) {
	// mustShowEnv("$env:path")
	// mustEnvEq("$env:some", "value")

	mustRegEq(`HKCU\Software\mh-cbon\hello`, "Version", "some version")
	mustRegEq(`HKCU\Software\mh-cbon\hello`, "InstallDir", `C:\Program Files\hello`)

	readDir("C:/Program Files/hello")
	readDir("C:/Program Files/hello/assets")
	readDir("C:/ProgramData/Microsoft/Windows/Start Menu/Programs/hello")

	mgr, svc := mustHaveWindowsService(service)
	defer mgr.Disconnect()
	mustHaveStartedWindowsService(service, svc)
	mustSucceed(svc.Close(), "Failed to close the service %v")

	// helloExecPath := "C:/Program Files/hello/hello.exe"
	// mustExecHello(helloExecPath, url)
	mustQueryHello(url, version)
	// mustStopWindowsService(svcName, svc)
}

func mustNotBeInstalled() {
	mustNotHaveWindowsService(service)
	mustNotQueryHello(url)

	mustNoDir("C:/Program Files/hello")
	mustNoDir("C:/ProgramData/Microsoft/Windows/Start Menu/Programs/hello")

	// mustShowEnv("$env:path")
	mustEnvEq("$env:some", "")

	mustNoReg(`HKCU\Software\mh-cbon`)
	mustNoReg(`HKLM\Software\mh-cbon`)
	mustNoReg(`HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\hello`)
	mustNoReg(`HKLM\Software\Microsoft\Windows\CurrentVersion\Uninstall\hello`)
}

func mustHaveWindowsService(n string) (*mgr.Mgr, *mgr.Service) {
	mgr, err := mgr.Connect()
	mustSucceed(err, "Failed to connect to the service manager %v")
	s, err := mgr.OpenService(n)
	mustSucceed(err, "Failed to open the service %v")
	if s == nil {
		mustSucceed(err, "Failed to find the service %v")
	}
	log.Printf("SUCCESS: Service %q exists\n", n)
	return mgr, s
}

func mustNotHaveWindowsService(n string) {
	mgr, err := mgr.Connect()
	mustSucceed(err, "Failed to connect to the service manager %v")
	defer mgr.Disconnect()
	s, err := mgr.OpenService(n)
	mustFail(err, "Must fail to open the service")
	if s == nil {
		mustFail(err, "Must fail to find the service")
	} else {
		defer s.Close()
	}
	log.Printf("SUCCESS: Service %q does not exist\n", n)
}

func mustHaveStartedWindowsService(n string, s *mgr.Service) {
	status, err := s.Query()
	mustSucceed(err, "Failed to query the service status %v")
	if status.State != svc.Running {
		mustSucceed(fmt.Errorf("Service not started %v", n))
	}
	log.Printf("SUCCESS: Service %q was started\n", n)
}

func mustStopWindowsService(n string, s *mgr.Service) {
	status, err := s.Control(svc.Stop)
	mustSucceed(err, "Failed to control the service status %v")
	if status.State != svc.Stopped {
		mustSucceed(fmt.Errorf("Service not stopped %v", n))
	}
	log.Printf("SUCCESS: Service %q was stopped\n", n)
}

func mustQueryHello(u, v string) {
	res := getURL(u)
	mustExec(res, "HTTP request failed %v")
	mustEqStdout(res, "["+v+"] hello, world\n", "Invalid HTTP response got=%q, want=%q")
	log.Printf("SUCCESS: Hello service query %q succeed\n", u)
}
func mustNotQueryHello(u string) {
	res := getURL(u)
	mustFail(res.Exec(), "HTTP request succeeded")
}

func confirm(err error, message string) {
	if err == nil {
		log.Printf("DONE: %v\n", message)
	} else {
		log.Printf("NOT-DONE: (%v) %v", err, message)
	}
}
func mustSucceed(err error, format ...string) {
	if err != nil {
		if len(format) > 0 {
			err = fmt.Errorf(format[0], err)
		}
		log.Fatal(err)
	}
}
func mustSucceedDetailed(err error, e interface{}, format ...string) {
	if x, ok := e.(stdouter); ok {
		fmt.Printf("%T:%v\n", x, x.Stdout())
	}
	if x, ok := e.(stderrer); ok {
		fmt.Printf("%T:%v\n", x, x.Stderr())
	}
	if err != nil {
		if len(format) > 0 {
			err = fmt.Errorf(format[0], err)
		} else {
			err = fmt.Errorf("%v", err)
		}
		log.Fatal(err)
	}
}
func mustFail(err error, format ...string) {
	if err == nil {
		msg := "Expected to fail"
		if len(format) > 0 {
			msg = format[0]
		}
		log.Fatal(msg)
	}
}

func mustContain(s, substr string) {
	if strings.Index(s, substr) == -1 {
		log.Println(s)
		log.Fatalf("Failed to find %q", substr)
	}
}
func mustShowEnv(e string) {
	psShowEnv := makeCmd("PowerShell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", e)
	mustExec(psShowEnv, "powershell command failed %v")
	log.Printf("showEnv ok %v %q", e, psShowEnv.Stdout())
}
func mustShowReg(key, value string) {
	cmd := makeCmd("reg", "query", key, "/v", value)
	mustExec(cmd, "registry query command failed %v")
	log.Printf(`showReg ok %v\%v %q`, key, value, cmd.Stdout())
}
func maybeShowEnv(e string) *cmdExec {
	psShowEnv := makeCmd("PowerShell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", e)
	warnExec(psShowEnv, "powershell command failed %v")
	log.Printf("showEnv ok %v %q", e, psShowEnv.Stdout())
	return psShowEnv
}
func mustChdir(s fmt.Stringer) {
	path := s.String()
	mustSucceed(os.Chdir(path), fmt.Sprintf("chDir failed %q\n%%v", path))
	log.Printf("chdir ok %v", path)
}
func mustEnvEq(env string, expect string, format ...string) {
	c := maybeShowEnv(env)
	got := c.Stdout()
	if len(got) > 0 {
		got = got[0 : len(got)-2]
	}
	f := fmt.Sprintf("Env %q is not equal to want=%q, got=%q", env, expect, got)
	mustSucceed(isTrue(got == expect, f))
	log.Printf("mustEnvEq ok %v=%q", env, expect)
}
func mustRegEq(key, value, expected string) {
	cmd := makeCmd("reg", "query", key, "/v", value)
	mustExec(cmd, "registry query command failed %v")
	mustContain(cmd.Stdout(), expected)
}
func mustNoReg(key string) {
	cmd := makeCmd("reg", "query", key)
	mustFail(cmd.Exec(), "registry query command succeeded %v, \n%v", cmd.Stdout())
}
func mustContains(path fmt.Stringer, file string) {
	s := mustLs(path)
	_, ex := s[file]
	f := fmt.Sprintf("File %q not found in %q", file, path)
	mustSucceed(isTrue(ex, f))
	log.Printf("mustContains ok %v %v", path, file)
}
func mustLs(s fmt.Stringer) map[string]os.FileInfo {
	ret := make(map[string]os.FileInfo)
	path := s.String()
	files, err := ioutil.ReadDir(path)
	mustSucceed(err, fmt.Sprintf("readdir failed %q, err=%%v", s))
	for _, f := range files {
		ret[f.Name()] = f
	}
	return ret
}

type starter interface {
	Start() error
}

func mustStart(e starter, format ...string) {
	if len(format) < 1 {
		format = []string{"Start err: %v"}
	}
	mustSucceed(e.Start(), format[0])
}

type waiter interface {
	Wait() error
}

func mustWait(e waiter, format ...string) {
	if len(format) < 1 {
		format = []string{"Wait err: %v"}
	}
	mustSucceed(e.Wait(), format[0])
}

type execer interface {
	Exec() error
}

func mustExec(e execer, format ...string) {
	if len(format) < 1 {
		format = []string{"Exec err: %v"}
	}
	mustSucceedDetailed(e.Exec(), e, format[0])
	log.Printf("mustExec success %v", e)
}

func warnExec(e execer, format ...string) {
	if err := e.Exec(); err != nil {
		if len(format) < 1 {
			format = []string{"Exec err: %v"}
		}
		log.Printf(format[0], err)
	}
}

type killer interface {
	Kill() error
}

func mustKill(e killer, format ...string) {
	if len(format) < 1 {
		format = []string{"Kill err: %v"}
	}
	mustSucceed(e.Kill(), format[0])
}

type exister interface {
	exists() bool
}

func mustExist(file string, format ...string) {
	e := makeFile(file)
	if len(format) < 1 {
		format = []string{fmt.Sprintf("mustExist err: %T does not exist %q, got %%v", e, e)}
	}
	mustSucceed(isTrue(e.exists(), format[0]))
}

func isTrue(b bool, format ...string) error {
	if b {
		return nil
	}
	if len(format) < 1 {
		format = []string{"isTrue got %v"}
	}
	return fmt.Errorf(format[0], b)
}

type stderrer interface {
	Stderr() string
}

type stdouter interface {
	Stdout() string
}

func mustEqStdout(e stdouter, expected string, format ...string) {
	got := e.Stdout()
	if len(format) < 1 {
		format = []string{"mustEqStdout failed: output does not match, got=%q, want=%q: "}
	}
	mustSucceed(isTrue(got == expected), fmt.Sprintf(format[0], got, expected))
}

func makeFile(f string) *file {
	return &file{f}
}

type file struct {
	path string
}

func (f *file) exists() bool {
	if _, err := os.Stat(f.path); os.IsNotExist(err) {
		return false
	}
	return true
}
func (f *file) String() string {
	return f.path
}

func makeDir(f string) *dir {
	return &dir{f}
}

type dir struct {
	path string
}

func (d *dir) exists() bool {
	if _, err := os.Stat(d.path); os.IsNotExist(err) {
		return false
	}
	return true
}
func (d *dir) String() string {
	return d.path
}

func getURL(url string) *httpRequest {
	return &httpRequest{url: url}
}

type httpRequest struct {
	url        string
	body       string
	statusCode int
	headers    map[string][]string
}

func (f *httpRequest) String() string {
	return f.url
}

func (f *httpRequest) Stdout() string {
	return f.body
}
func (f *httpRequest) Header(name string) []string {
	return f.headers[name]
}
func (f *httpRequest) RespondeCode() int {
	return f.statusCode
}
func (f *httpRequest) ExitOk() bool {
	return f.statusCode == 200
}
func (f *httpRequest) Exec() error {
	response, err := http.Get(f.url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	f.headers = response.Header
	f.statusCode = response.StatusCode
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	f.body = string(b)
	return nil
}

type cmdExec struct {
	*exec.Cmd
	bin        string
	args       []string
	startErr   error
	hasStarted bool
	waitErr    error
	hasWaited  bool
	stdout     *bytes.Buffer
	stderr     *bytes.Buffer
}

func (e *cmdExec) String() string {
	return e.bin + " " + strings.Join(e.args, " ")
}

func (e *cmdExec) SetArgs(args []string) error {
	if e.hasStarted {
		return fmt.Errorf("Cannot set arguments on command already started")
	}
	if e.hasWaited {
		return fmt.Errorf("Cannot set arguments on command already waited")
	}
	e.args = args
	return nil
}

func (e *cmdExec) Start() error {
	if !e.hasStarted {
		e.startErr = e.Cmd.Start()
	}
	e.hasStarted = true
	return e.startErr
}
func (e *cmdExec) Wait() error {
	if !e.hasWaited {
		e.waitErr = e.Cmd.Wait()
	}
	e.hasWaited = true
	return e.waitErr
}
func (e *cmdExec) Exec() error {
	if err := e.Start(); err != nil {
		return err
	}
	return e.Wait()
}
func (e *cmdExec) Kill() error {
	return e.Process.Kill()
}
func (e *cmdExec) Stdout() string {
	if e.hasStarted == false {
		log.Fatal("Process must have run")
	}
	return e.stdout.String()
}
func (e *cmdExec) Stderr() string {
	if e.hasStarted == false {
		log.Fatal("Process must have run")
	}
	return e.stderr.String()
}
func (e *cmdExec) ExitOk() bool {
	if e.hasWaited == false {
		log.Fatal("Process must have run")
	}
	return e.ProcessState.Exited() && e.ProcessState.Success()
}

func makeCmd(w string, a ...string) *cmdExec {
	log.Printf("makeCmd: %v %v\n", w, a)
	cmd := exec.Command(w, a...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	return &cmdExec{Cmd: cmd, stdout: &stdout, stderr: &stderr}
}

func readDir(s string) {
	files, err := ioutil.ReadDir(s)
	mustSucceed(err, fmt.Sprintf("readdir failed %q, err=%%v", s))
	log.Printf("Content of directory %q\n", s)
	for _, f := range files {
		log.Printf("    %v\n", f.Name())
	}
}

func mustNoDir(s string) {
	_, err := os.Stat(s)
	mustFail(err, fmt.Sprintf("directory %q exists", s))
}

func readFile(s string) {
	fd, err := os.Open(s)
	mustSucceed(err, fmt.Sprintf("readfile failed %q, err=%%v", s))
	defer fd.Close()
	if fd != nil {
		io.Copy(fd, os.Stdout)
	}
}

func rmFile(s string) error {
	return os.Remove(s)
}

func readLog(filename string) string {
	b, err := readFileUTF16(filename)
	mustSucceed(err, fmt.Sprintf("readlog failed %q, err=%%v", filename))
	return string(b)
}

func readFileUTF16(filename string) ([]byte, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	win16be := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	utf16bom := unicode.BOMOverride(win16be.NewDecoder())
	unicodeReader := transform.NewReader(bytes.NewReader(raw), utf16bom)
	decoded, err := ioutil.ReadAll(unicodeReader)
	return decoded, err
}
