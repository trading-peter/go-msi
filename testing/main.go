//go:build windows
// +build windows

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func main() {
	mustNotBeInstalled()

	gopath := os.Getenv("GOPATH")
	wd := filepath.Join(gopath, "src/github.com/observiq/go-msi/testing/hello")
	mustContainFile(wd, "hello.go")
	mustChdir(wd)

	helloBuild := makeCmd("go", "build", "-o", "build/amd64/hello.exe", "hello.go")
	mustExec(helloBuild, "hello build failed %v")

	setGuid := makeCmd("C:/go-msi/go-msi.exe", "set-guid")
	mustExec(setGuid, "Packaging set-guid failed %v")

	setFiles := makeCmd("C:/go-msi/go-msi.exe", "add-files", "--dir", "some", "--includes", "globbed/**", "--excludes", "**/file3")
	mustExec(setFiles, "Packaging set-files failed %v")

	msi := "hello.msi"
	version := "12.34.5678"
	pkg := makeCmd("C:/go-msi/go-msi.exe", "make",
		"--msi", msi,
		"--version", version,
		"--arch", "amd64",
		"--property", "SOME_VERSION=some version",
		"--keep",
	)
	mustExec(pkg, "Packaging failed %v")
	mustExist(msi, "Package file is missing %v")

	packageInstall := makeCmd("msiexec", "/i", msi, "/q", "/l*v", "log-install.txt", "MINIMUMBUILD=0")
	mustFail(packageInstall.Exec(), "Package installation succeeded")
	content := readLog("log-install.txt")
	mustContain(content, "Product: hello -- some condition message")
	mustContain(content, "Product: hello -- Installation failed")
	mustSucceed(rmFile("log-install.txt"), "rmfile failed %v")

	install(msi, version, true, true, true)

	packageReinstall := makeCmd("msiexec", "/i", msi, "/q", "/l*v", "log-install.txt")
	mustExec(packageReinstall, "Package re-installation failed %v")
	content = readLog("log-install.txt")
	mustContain(content, "Product: hello -- Configuration completed successfully.")
	mustContain(content, "Windows Installer reconfigured the product. Product Name: hello. Product Version: "+version+". Product Language: 1033. Manufacturer: mh-cbon.")
	mustBeInstalled(version, true, true, true)
	mustSucceed(rmFile("log-install.txt"), "rmfile failed %v")

	oldMsi := msi
	msi = "hello-2.msi"
	pkg = makeCmd("C:/go-msi/go-msi.exe", "make",
		"--msi", msi,
		"--version", "v12.34.5677", // version down
		"--arch", "amd64",
		"--property", "SOME_VERSION=some version",
		"--keep",
	)
	mustExec(pkg, "Packaging failed %v")
	mustExist(msi, "Package file is missing %v")
	packageDowngrade := makeCmd("msiexec", "/i", msi, "/q", "/l*v", "log-install.txt")
	mustFail(packageDowngrade.Exec(), "Package downgrade succeeded")
	content = readLog("log-install.txt")
	mustContain(content, "Product: hello -- A newer version of this software is already installed.")
	mustBeInstalled(version, true, true, true)
	mustSucceed(rmFile("log-install.txt"), "rmfile failed %v")

	version = "12.34.5679" // version up
	pkg = makeCmd("C:/go-msi/go-msi.exe", "make",
		"--msi", msi,
		"--version", version,
		"--arch", "amd64",
		"--property", "SOME_VERSION=some version",
		"--keep",
	)
	mustExec(pkg, "Packaging failed %v")
	mustExist(msi, "Package file is missing %v")
	install(msi, version, true, true, true)

	packageUninstall := makeCmd("msiexec", "/x", oldMsi, "/q", "/l*v", "log-uninstall.txt")
	mustFail(packageUninstall.Exec(), "Old package uninstall succeeded")
	content = readLog("log-uninstall.txt")
	mustContain(content, "This action is only valid for products that are currently installed")
	mustBeInstalled(version, true, true, true)
	mustSucceed(rmFile("log-uninstall.txt"), "rmfile failed %v")

	uninstall(msi, version)

	install(msi, version, false, true, true, "STARTMENUSHORTCUT=no")
	uninstall(msi, version)

	install(msi, version, true, false, true, "DESKTOPSHORTCUT=no")
	uninstall(msi, version)

	install(msi, version, true, true, false, "ENVVAR=no")
	uninstall(msi, version)

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
		"-c", "changelog ghrelease --version "+version,
		"--keep",
	)
	mustExec(chocoPkg, "hello choco package make failed %v")
	mustExist("hello."+version+".nupkg", "Chocolatey nupkg file is missing %v")

	chocoInstall := makeCmd("choco", "install", "hello."+version+".nupkg", "-y")
	mustExec(chocoInstall, "hello choco package install failed %v")

	mustBeInstalled(version, true, true, true)

	chocoUninstall := makeCmd("choco", "uninstall", "hello", "-v", "-d", "-y", "--force")
	mustExec(chocoUninstall, "hello choco package uninstall failed %v")

	mustNotBeInstalled()

	fmt.Println("\nSuccess!")
}

var hookFile = filepath.Join(homedir(), "hook.txt")

func homedir() string {
	usr, err := user.Current()
	mustSucceed(err)
	return usr.HomeDir
}

func install(msi, version string, menuShortcut, desktopShortcut, envvar bool, properties ...string) {
	packageInstall := makeCmd("msiexec", append([]string{"/i", msi, "/q", "/l*v", "log-install.txt"}, properties...)...)
	mustExec(packageInstall, "Package installation failed %v")
	content := readLog("log-install.txt")
	mustContain(content, "Product: hello -- Installation completed successfully.")
	mustContain(content, "Windows Installer installed the product. Product Name: hello. Product Version: "+version+". Product Language: 1033. Manufacturer: mh-cbon.")
	mustBeInstalled(version, menuShortcut, desktopShortcut, envvar)
	mustSucceed(rmFile("log-install.txt"), "rmfile failed %v")
	mustEqual(strings.TrimSpace(readFile(hookFile)), "hook")
	mustSucceed(rmFile(hookFile), "rmfile failed %v")
}

func uninstall(msi, version string) {
	packageUninstall := makeCmd("msiexec", "/x", msi, "/q", "/l*v", "log-uninstall.txt")
	mustExec(packageUninstall, "Package uninstallation failed %v")
	content := readLog("log-uninstall.txt")
	mustContain(content, "Product: hello -- Removal completed successfully.")
	mustContain(content, "Windows Installer removed the product. Product Name: hello. Product Version: "+version+". Product Language: 1033. Manufacturer: mh-cbon.")
	mustNotBeInstalled()
	mustSucceed(rmFile("log-uninstall.txt"), "rmfile failed %v")
	mustEqual(strings.TrimSpace(readFile(hookFile)), "hook")
	mustSucceed(rmFile(hookFile), "rmfile failed %v")
}

const (
	url                     = "http://localhost:8080/"
	service                 = "HelloSvc"
	menuShortcutLocation    = "C:/ProgramData/Microsoft/Windows/Start Menu/Programs/hello.lnk"
	desktopShortcutLocation = "C:/Users/Public/Desktop/hello.lnk"
)

func mustBeInstalled(version string, menuShortcut, desktopShortcut, envvar bool) {
	mustEnvSuffix("path", `C:\Program Files\hello\`)
	mustEnvEq("some", "value")
	if envvar {
		mustEnvEq("condition", "ok")
	} else {
		mustEnvEq("condition", "")
	}

	mustRegEq(`HKCU\Software\mh-cbon\hello`, "Version", "some version")
	mustRegEq(`HKCU\Software\mh-cbon\hello`, "InstallDir", `C:\Program Files\hello`)

	mustExist("C:/Program Files/hello", "Files missing %v")
	mustExist("C:/Program Files/hello/assets", "Directory missing %v")
	mustExist("C:/Program Files/hello/assets/dir1", "Directory missing %v")
	mustExist("C:/Program Files/hello/assets/dir1/file2", "File missing %v")
	mustExist("C:/Program Files/hello/assets/file1", "File missing %v")
	mustExist("C:/Program Files/hello/globbed", "Directory missing %v")
	mustExist("C:/Program Files/hello/globbed/file4", "File missing %v")
	mustNotExist("C:/Program Files/hello/globbed/dir1")
	if menuShortcut {
		mustExist(menuShortcutLocation, "Start menu shortcut is missing %v")
	} else {
		mustNotExist(menuShortcutLocation)
	}
	if desktopShortcut {
		mustExist(desktopShortcutLocation, "Desktop shortcut is missing %v")
	} else {
		mustNotExist(desktopShortcutLocation)
	}
	mustEqual(strings.TrimSpace(readFile("C:/Program Files/hello/install-hook.txt")), "install hook")
	mustEqual(strings.TrimSpace(readFile("C:/Program Files/hello/install-hook-with-passing-condition.txt")), "install hook with passing condition")

	mgr, svc := mustHaveWindowsService(service)
	defer mgr.Disconnect()
	mustHaveStartedWindowsService(service, version, svc)
	mustSucceed(svc.Close(), "Failed to close the service %v")

	mustQueryHello(url, version)
}

func mustNotBeInstalled() {
	mustNotHaveWindowsService(service)
	mustNotQueryHello(url)

	mustNotExist("C:/Program Files/hello")
	mustNotExist(menuShortcutLocation)
	mustNotExist(desktopShortcutLocation)

	mustEnvNotContain("path", `C:\Program Files\hello`)
	mustEnvEq("some", "")

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
}

func mustHaveStartedWindowsService(n, version string, s *mgr.Service) {
	status, err := s.Query()
	mustSucceed(err, "Failed to query the service status %v")
	if status.State != svc.Running {
		mustSucceed(fmt.Errorf("Service not started %v", n))
	}
	sc := makeCmd("sc", "qc", service)
	mustExec(sc, "sc failed %v")
	expected := `[SC] QueryServiceConfig SUCCESS

SERVICE_NAME: HelloSvc
        TYPE               : 10  WIN32_OWN_PROCESS 
        START_TYPE         : 2   AUTO_START  (DELAYED)
        ERROR_CONTROL      : 1   NORMAL
        BINARY_PATH_NAME   : "C:\Program Files\hello\hello.exe" ` + version + `
        LOAD_ORDER_GROUP   : 
        TAG                : 0
        DISPLAY_NAME       : Hello!
        DEPENDENCIES       : SENS
                           : ProfSvc
        SERVICE_START_NAME : LocalSystem
`
	mustEqualLines(sc.Stdout(), expected)
}

func mustEqualLines(actual, expected string) {
	actualLines := strings.Split(actual, "\n")
	expectedLines := strings.Split(expected, "\n")
	if len(actualLines) != len(expectedLines) {
		log.Fatalf("expected %s differs from actual %s", expected, actual)
	}
	for i, a := range actualLines {
		a = trimPath(a)
		b := trimPath(expectedLines[i])
		if a != b {
			log.Fatalf("expected %q differs from actual %q", b, a)
		}
	}
}

func trimPath(path string) string {
	return strings.TrimSpace(strings.Replace(path, "c:", "C:", -1))
}

func mustStopWindowsService(n string, s *mgr.Service) {
	status, err := s.Control(svc.Stop)
	mustSucceed(err, "Failed to control the service status %v")
	if status.State != svc.Stopped {
		mustSucceed(fmt.Errorf("Service not stopped %v", n))
	}
}

func mustQueryHello(u, v string) {
	res := getURL(u)
	mustExec(res, "HTTP request failed %v")
	mustEqStdout(res, "["+v+"] hello, world\n", "Invalid HTTP response got=%q, want=%q")
}
func mustNotQueryHello(u string) {
	res := getURL(u)
	mustFail(res.Exec(), "HTTP request succeeded")
}

func mustSucceed(err error, format ...string) {
	if err != nil {
		if len(format) > 0 {
			err = fmt.Errorf(format[0], err)
		}
		log.Fatal(err)
	}
}
func mustSucceedDetailed(err error, e execer, format string) {
	if err != nil {
		if len(format) > 0 {
			err = fmt.Errorf(format, err)
		}
		log.Print(e.Stderr())
		log.Print(e.Stdout())
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
		log.Fatalf("Failed to find %q", substr)
	}
}
func mustEqual(actual, expected string) {
	if expected != actual {
		log.Fatalf("expected %s differs from actual %s", expected, actual)
	}
}

func getEnv(env string) string {
	value := getEnvFrom(env, "User")
	if env == "path" || value == "" {
		value += getEnvFrom(env, "Machine")
	}
	return value
}
func getEnvFrom(env, hive string) string {
	cmd := makeCmd("PowerShell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", `[System.Environment]::GetEnvironmentVariable("`+env+`", "`+hive+`")`)
	err := cmd.Exec()
	if err != nil {
		log.Fatalf("getEnvFrom failed: %s", err)
	}
	return trimPath(cmd.Stdout())
}
func mustShowReg(key, value string) {
	cmd := makeCmd("reg", "query", key, "/v", value)
	mustExec(cmd, "registry query command failed %v")
}
func mustChdir(path string) {
	mustSucceed(os.Chdir(path), fmt.Sprintf("chDir failed %q\n%%v", path))
}
func mustEnvEq(env string, expect string, format ...string) {
	got := getEnv(env)
	if got != expect {
		log.Fatalf("Env %q is not equal to %q, got=%q", env, expect, got)
	}
}
func mustEnvSuffix(env string, expect string, format ...string) {
	got := getEnv(env)
	if !strings.HasSuffix(got, ";"+expect) {
		log.Fatalf("Env %q does not have suffix %q, got=%q", env, expect, got)
	}
}
func mustEnvNotContain(env string, expect string, format ...string) {
	got := getEnv(env)
	if strings.Index(got, expect) > 0 {
		log.Fatalf("Env %q contains %q, got=%q", env, expect, got)
	}
}
func mustRegEq(key, value, expected string) {
	cmd := makeCmd("reg", "query", key, "/v", value)
	mustExec(cmd, "registry query command failed %v")
	mustContain(trimPath(cmd.Stdout()), expected)
}
func mustNoReg(key string) {
	cmd := makeCmd("reg", "query", key)
	mustFail(cmd.Exec(), "registry query command succeeded %v, \n%v", cmd.Stdout())
}
func mustContainFile(path, file string) {
	s := mustLs(path)
	_, ex := s[file]
	f := fmt.Sprintf("File %q not found in %q", file, path)
	mustSucceed(isTrue(ex, f))
}
func mustLs(path string) map[string]os.FileInfo {
	ret := make(map[string]os.FileInfo)
	files, err := ioutil.ReadDir(path)
	mustSucceed(err, fmt.Sprintf("readdir failed %q, err=%%v", path))
	for _, f := range files {
		ret[f.Name()] = f
	}
	return ret
}

type execer interface {
	Exec() error
	Stderr() string
	Stdout() string
}

func mustExec(e execer, format string) {
	mustSucceedDetailed(e.Exec(), e, format)
}

func mustExist(file string, format ...string) {
	if len(format) < 1 {
		format = []string{fmt.Sprintf("mustExist err: %q does not exist, got %%v", file)}
	}
	_, err := os.Stat(file)
	mustSucceed(err, format[0])
}

func mustNotExist(file string) {
	_, err := os.Stat(file)
	if !os.IsNotExist(err) {
		log.Fatalf("mustNotExist err: %q exists", file)
	}
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

func mustEqStdout(e execer, expected string, format ...string) {
	got := e.Stdout()
	if len(format) < 1 {
		format = []string{"mustEqStdout failed: output does not match, got=%q, want=%q: "}
	}
	mustSucceed(isTrue(got == expected), fmt.Sprintf(format[0], got, expected))
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
func (f *httpRequest) Stderr() string {
	return ""
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

func (e *cmdExec) Start() error {
	if !e.hasStarted {
		fmt.Println("starting", e.Cmd.Path, e.Cmd.Args)
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

func makeCmd(w string, a ...string) *cmdExec {
	cmd := exec.Command(w, a...)
	if w == "msiexec" {
		// see https://github.com/golang/go/issues/15566
		cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: " " + strings.Join(a, " ")}
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	return &cmdExec{Cmd: cmd, stdout: &stdout, stderr: &stderr}
}

func readFile(filename string) string {
	b, err := ioutil.ReadFile(filename)
	mustSucceed(err, fmt.Sprintf("readfile failed %q, err=%%v", filename))
	return string(b)
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
