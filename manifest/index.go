package manifest

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/google/uuid"
)

// WixManifest is the struct to decode a wix.json file.
type WixManifest struct {
	Compression string  `json:"compression,omitempty"`
	Product     string  `json:"product"`
	Company     string  `json:"company"`
	Version     Version `json:"-"`
	License     string  `json:"license,omitempty"`
	Banner      string  `json:"banner,omitempty"`
	Dialog      string  `json:"dialog,omitempty"`
	Icon        string  `json:"icon,omitempty"`
	Info        *Info   `json:"info,omitempty"`
	UpgradeCode string  `json:"upgrade-code"`
	Directory
	Environments []Environment  `json:"environments,omitempty"`
	Registries   []RegistryItem `json:"registries,omitempty"`
	Shortcuts    []Shortcut     `json:"shortcuts,omitempty"`
	Choco        ChocoSpec      `json:"choco"`
	Hooks        []Hook         `json:"hooks,omitempty"`
	Properties   []Property     `json:"properties,omitempty"`
	Conditions   []Condition    `json:"conditions,omitempty"`
}

// Version stores version related data in various formats.
type Version struct {
	User    string
	Display string
	MSI     string
	Hex     int64
}

// Info lists the control panel program information.
// Each member data is named after the matching column name in the uninstall
// program list.
type Info struct {
	Comments         string `json:"comments,omitempty"`
	Contact          string `json:"contact,omitempty"`
	HelpLink         string `json:"help-link,omitempty"`
	SupportTelephone string `json:"support-telephone,omitempty"`
	SupportLink      string `json:"support-link,omitempty"`
	UpdateInfoLink   string `json:"update-info-link,omitempty"`
	Readme           string `json:"readme,omitempty"`
	Size             int64  `json:"-"` // in kilobytes
}

// File is the struct to decode a file.
type File struct {
	ID             int      `json:"-"`
	Path           string   `json:"path,omitempty"`
	Service        *Service `json:"service,omitempty"`
	NeverOverwrite bool     `json:"never_overwrite,omitempty"`
	Permanent      bool     `json:"permanent,omitempty"`
}

// Directory stores a list of files and a list of sub-directories.
type Directory struct {
	ID          int         `json:"-"`
	Name        string      `json:"name,omitempty"`
	Files       []File      `json:"files,omitempty"`
	Directories []Directory `json:"directories,omitempty"`
}

type fileWalker func(file File) (File, error)

func (dir *Directory) walkFiles(f fileWalker) error {
	var err error
	dir.Files, dir.Directories, err = walkFiles(dir.Files, dir.Directories, f)
	return err
}

func walkFiles(files []File, dirs []Directory, f fileWalker) ([]File, []Directory, error) {
	for i, file := range files {
		var err error
		if files[i], err = f(file); err != nil {
			return files, dirs, err
		}
	}
	for i := range dirs {
		if err := dirs[i].walkFiles(f); err != nil {
			return files, dirs, err
		}
	}
	return files, dirs, nil
}

type directoryWalker func(dir Directory) (Directory, error)

func (dir *Directory) walkDirectories(f directoryWalker) error {
	var err error
	dir.Directories, err = walkDirectories(dir.Directories, f)
	return err
}

func walkDirectories(dirs []Directory, f directoryWalker) ([]Directory, error) {
	for i := range dirs {
		var err error
		dirs[i], err = f(dirs[i])
		if err != nil {
			return dirs, err
		}
		if err := dirs[i].walkDirectories(f); err != nil {
			return dirs, err
		}
	}
	return dirs, nil
}

// Service is the struct to decode a service.
type Service struct {
	Name         string   `json:"name"`
	Bin          string   `json:"-"`
	Start        string   `json:"start"`
	Delayed      bool     `json:"-"`
	DisplayName  string   `json:"display-name,omitempty"`
	Description  string   `json:"description,omitempty"`
	Arguments    string   `json:"arguments,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// ChocoSpec is the struct to decode the choco key of a wix.json file.
type ChocoSpec struct {
	ID             string `json:"id,omitempty"`
	Title          string `json:"title,omitempty"`
	Authors        string `json:"authors,omitempty"`
	Owners         string `json:"owners,omitempty"`
	Description    string `json:"description,omitempty"`
	ProjectURL     string `json:"project-url,omitempty"`
	Tags           string `json:"tags,omitempty"`
	LicenseURL     string `json:"license-url,omitempty"`
	IconURL        string `json:"icon-url,omitempty"`
	RequireLicense bool   `json:"require-license,omitempty"`
	MsiFile        string `json:"-"`
	MsiSum         string `json:"-"`
	BuildDir       string `json:"-"`
	ChangeLog      string `json:"-"`
}

// Hook describes a command to run on install / uninstall.
type Hook struct {
	Command       string `json:"command,omitempty"`
	CookedCommand string `json:"-"`
	When          string `json:"when,omitempty"`
	Return        string `json:"return,omitempty"`
	Condition     string `json:"condition,omitempty"`
	Impersonate   string `json:"impersonate,omitempty"`
	Execute       string `json:"execute,omitempty"`
}

// Property describes a property to initialize.
type Property struct {
	ID       string    `json:"id"`
	Registry *Registry `json:"registry,omitempty"`
	Value    *Value    `json:"value,omitempty"`
}

// Registry describes a registry entry.
type Registry struct {
	Path string `json:"path"`
	Root string `json:"-"`
	Key  string `json:"-"`
	Name string `json:"name,omitempty"`
}

// Value describes a simple string value
type Value string

// Condition describes a condition to check before installation.
type Condition struct {
	Condition string `json:"condition"`
	Message   string `json:"message"`
}

// Environment is the struct to decode environment variables of the wix.json file.
type Environment struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Permanent string `json:"permanent"`
	System    string `json:"system"`
	Action    string `json:"action"`
	Part      string `json:"part"`
	Condition string `json:"condition,omitempty"`
}

// Shortcut is the struct to decode shortcut value of the wix.json file.
type Shortcut struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Location    string             `json:"location"`
	Target      string             `json:"target"`
	WDir        string             `json:"wdir,omitempty"`
	Arguments   string             `json:"arguments,omitempty"`
	Icon        string             `json:"icon,omitempty"`
	Condition   string             `json:"condition,omitempty"`
	Properties  []ShortcutProperty `json:"properties,omitempty"`
}

// ShortcutProperty stands for a key value association.
type ShortcutProperty struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RegistryItem is the struct to decode a registry item.
type RegistryItem struct {
	Registry
	Values    []RegistryValue `json:"values,omitempty"`
	Condition string          `json:"condition,omitempty"`
}

// RegistryValue is the struct to decode a registry value.
type RegistryValue struct {
	Name  string `json:"name"`
	Type  string `json:"type,omitempty"` // string (default if omitted), integer, ...
	Value string `json:"value"`
}

// Write the manifest to the given file,
// if file is empty, writes to wix.json
func (wixFile *WixManifest) Write(p string) error {
	if p == "" {
		p = "wix.json"
	}
	byt, err := json.MarshalIndent(wixFile, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(p, byt, 0644)
	if err != nil {
		return err
	}
	return nil
}

// Load the manifest from given file path,
// if the file path is empty, reads from wix.json
func (wixFile *WixManifest) Load(p string) error {
	if p == "" {
		p = "wix.json"
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		return err
	}

	dat, err := ioutil.ReadFile(p)
	if err != nil {
		return fmt.Errorf("JSON ReadFile failed with %v", err)
	}

	err = json.Unmarshal(dat, &wixFile)
	if err != nil {
		return fmt.Errorf("JSON Unmarshal failed with %v", err)
	}

	// dynamically build wixFile.Directories
	return wixFile.buildDirectoriesRecursive()
}

// buildDirectoriesRecursive detects all files and directories nested under a top level
// directory. The wix.JSON should look similar to:
//
//	 "directories": [
//	  	{
//				"name": "config"
//		    },
//		    {
//				"name": "launcher"
//		    }
//		],
func (wixFile *WixManifest) buildDirectoriesRecursive() error {
	for key, _ := range wixFile.Directories {
		err := buildDirectories(".", &wixFile.Directories[key])
		if err != nil {
			return err
		}
	}

	// Write the dynamic config to disk so we can inspect it manually
	// while debugging builds.
	b, err := json.MarshalIndent(wixFile, "", " ")
	if err != nil {
		return fmt.Errorf("DEBUG: failed to marshal fix config: %v", err)
	}
	f, err := os.Create("wix.dynamic.json")
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return fmt.Errorf("failed to write dynamic wix config to disk: %v", err)
	}
	return f.Close()
}

// buildDirectories detects all sub files and directories for the given
// *Directory. Useful for performing auto detection when there are too
// many files and directories to specify in the wix.json.
func buildDirectories(parentPath string, dir *Directory) error {
	// Get a list of all files and directories
	p := path.Join(parentPath, dir.Name)
	list, err := ioutil.ReadDir(p)
	if err != nil {
		return fmt.Errorf("failed to read path: %s", p)
	}

	// Iterate over the list:
	// Append all files to the directories []Files list
	// When a directory is detected, call this function again
	// and grap it's sub directories and files before appending
	// to the parent directory.
	for _, sub := range list {
		// purposely shadowing
		parentPath := path.Join(parentPath, dir.Name)

		if !sub.IsDir() {
			subFile := File{
				Path: path.Join(parentPath, sub.Name()),
			}
			dir.Files = append(dir.Files, subFile)
			continue
		}

		subDir := Directory{
			Name: sub.Name(),
		}

		err := buildDirectories(parentPath, &subDir)
		if err != nil {
			return err
		}

		// When finished building the directory tree, append the directory
		// to the parent directory.
		dir.Directories = append(dir.Directories, subDir)
	}

	return nil
}

func (wixFile *WixManifest) check() error {
	for _, hook := range wixFile.Hooks {
		switch hook.When {
		case "install", "uninstall", "":
		default:
			return fmt.Errorf(`Invalid "when" value in hook: %s`, hook.When)
		}
		switch hook.Impersonate {
		case "yes", "no":
		default:
			return fmt.Errorf(`Invalid "impersonate" value in hook: %s`, hook.Impersonate)
		}
	}
	for _, shortcut := range wixFile.Shortcuts {
		switch shortcut.Location {
		case "program", "desktop":
		default:
			return fmt.Errorf(`Invalid "location" value in shortcut: %s`, shortcut.Location)
		}
	}
	if wixFile.NeedGUID() {
		return fmt.Errorf(`The manifest needs Guid, To update your file automatically run "go-msi set-guid"`)
	}
	return nil
}

// SetGuids generates and apply guid values appropriately
func (wixFile *WixManifest) SetGuids(force bool) (bool, error) {
	updated := false
	if wixFile.UpgradeCode == "" || force {
		guid, err := makeGUID()
		if err != nil {
			return updated, err
		}
		wixFile.UpgradeCode = guid
		updated = true
	}
	return updated, nil
}

func makeGUID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return strings.ToUpper(id.String()), nil
}

// NeedGUID tells if the manifest json file is missing guid values.
func (wixFile *WixManifest) NeedGUID() bool {
	return wixFile.UpgradeCode == ""
}

// RewriteFilePaths reads files and directories of the wix.json file
// and turn their values into a relative path to out
// where out is the path to the wix templates files.
func (wixFile *WixManifest) RewriteFilePaths(out string) error {
	var err error
	out, err = filepath.Abs(out)
	if err != nil {
		return err
	}
	if wixFile.License != "" {
		path, err := rewrite(out, wixFile.License)
		if err != nil {
			return err
		}
		wixFile.License = path
	}

	id := 1
	if err := wixFile.walkDirectories(func(dir Directory) (Directory, error) {
		dir.ID = id
		id++
		return dir, nil
	}); err != nil {
		return err
	}
	id = 1
	if err := wixFile.walkFiles(func(file File) (File, error) {
		path, err := rewrite(out, file.Path)
		if err != nil {
			return file, err
		}
		file.Path = path
		file.ID = id
		id++
		return file, nil
	}); err != nil {
		return err
	}
	for i, s := range wixFile.Shortcuts {
		if s.Icon != "" {
			path, err := rewrite(out, s.Icon)
			if err != nil {
				return err
			}
			wixFile.Shortcuts[i].Icon = path
		}
	}
	return nil
}

func rewrite(out, path string) (string, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Rel(out, filepath.ToSlash(path))
}

func validateCompression(wixFile *WixManifest) error {
	if wixFile.Compression == "" {
		return nil
	}
	compressions := []string{"high", "low", "medium", "mszip", "none"}
	for _, c := range compressions {
		if c == wixFile.Compression {
			return nil
		}
	}
	return fmt.Errorf("invalid compression %q, must be one of %s", wixFile.Compression, strings.Join(compressions, ", "))
}

// Normalize appropriately fixes some values within the decoded json.
// It applies defaults values on the wix/msi property to generate the msi package.
// It applies defaults values on the choco property to generate a nuget package.
func (wixFile *WixManifest) Normalize() error {
	if err := validateCompression(wixFile); err != nil {
		return err
	}

	if wixFile.Version.Display == "" {
		wixFile.Version.Display = wixFile.Version.User
	}
	// Wix version Field of Product element
	// does not support semver strings
	// it supports only something like x.x.x.x
	// So, if the version has metadata/prerelease values,
	// lets get ride of those and save the workable version
	// into Version.MSI field
	if n, err := strconv.ParseInt(wixFile.Version.User, 10, 64); err == nil {
		major := n >> 24
		minor := (n - major<<24) >> 16
		build := n - major<<24 - minor<<16
		wixFile.Version.MSI = fmt.Sprintf("%d.%d.%d", major, minor, build)
		wixFile.Version.Hex = n
	} else if v, err := semver.NewVersion(wixFile.Version.User); err == nil {
		if v.Major() > 255 || v.Minor() > 255 || v.Patch() > 65535 {
			return fmt.Errorf("Failed to parse version '%v', fields must not exceed maximum values of 255.255.65535", wixFile.Version.User)
		}

		if n, err := strconv.ParseInt(v.Metadata(), 10, 64); err == nil {
			// append a metadata version number if present
			wixFile.Version.MSI = fmt.Sprintf("%d.%d.%d.%d", v.Major(), v.Minor(), v.Patch(), n)
		} else {
			// only use major, minor, and patch if no numeric metadata
			wixFile.Version.MSI = fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
		}

		wixFile.Version.Hex = v.Major()<<24 + v.Minor()<<16 + v.Patch()
	} else {
		return fmt.Errorf("Failed to parse version '%v', must be either a semantic version or a single build/revision number", wixFile.Version.User)
	}

	if wixFile.Banner != "" {
		path, err := filepath.Abs(wixFile.Banner)
		if err != nil {
			return err
		}
		wixFile.Banner = path
	}
	if wixFile.Dialog != "" {
		path, err := filepath.Abs(wixFile.Dialog)
		if err != nil {
			return err
		}
		wixFile.Dialog = path
	}
	if wixFile.Icon != "" {
		path, err := filepath.Abs(wixFile.Icon)
		if err != nil {
			return err
		}
		wixFile.Icon = path
	}

	// choco fix
	if wixFile.Choco.ID == "" {
		wixFile.Choco.ID = wixFile.Product
	}
	if wixFile.Choco.Title == "" {
		wixFile.Choco.Title = wixFile.Product
	}
	if wixFile.Choco.Authors == "" {
		wixFile.Choco.Authors = wixFile.Company
	}
	if wixFile.Choco.Owners == "" {
		wixFile.Choco.Owners = wixFile.Company
	}
	if wixFile.Choco.Description == "" {
		wixFile.Choco.Description = wixFile.Product
	}
	wixFile.Choco.Tags += " admin" // required to pass chocolatey validation..

	for i, hook := range wixFile.Hooks {
		command, err := escapeHook(hook.Command)
		if err != nil {
			return err
		}
		if hook.Execute == "" {
			hook.Execute = "deferred"
		}
		if hook.Impersonate == "" {
			hook.Impersonate = "no"
			if hook.Execute == "immediate" {
				hook.Impersonate = "yes"
			}
		}
		hook.CookedCommand = command
		wixFile.Hooks[i] = hook
	}

	var err error
	// Split registry path into root and key
	for _, prop := range wixFile.Properties {
		reg := prop.Registry
		if reg != nil {
			if reg.Root, reg.Key, err = extractRegistry(reg.Path); err != nil {
				return err
			}
		}
	}
	for i := range wixFile.Registries {
		r := &wixFile.Registries[i]
		if r.Root, r.Key, err = extractRegistry(r.Path); err != nil {
			return err
		}
		for j := range r.Values {
			v := &r.Values[j]
			if v.Type == "" {
				v.Type = "string"
			}
		}
	}

	// Bind services to their file component
	if err := wixFile.walkFiles(func(file File) (File, error) {
		if file.Service != nil {
			file.Service.Bin = filepath.Base(file.Path)
			if file.Service.Start == "delayed" {
				file.Service.Start = "auto"
				file.Service.Delayed = true
			}
		}
		return file, nil
	}); err != nil {
		return err
	}

	// Compute install size
	var size int64
	if err := wixFile.walkFiles(func(file File) (File, error) {
		info, err := os.Stat(file.Path)
		if err != nil {
			return file, err
		}
		size += info.Size()
		return file, nil
	}); err != nil {
		return err
	}
	wixFile.Info.Size = size >> 10

	return wixFile.check()
}

func escapeHook(command string) (string, error) {
	cmd := strings.Trim(command, " ")
	if len(cmd) > 0 && cmd[0] != '"' {
		words := strings.Split(cmd, " ")
		cmd = `"` + words[0] + `"` + cmd[len(words[0]):]
	}
	buf := &bytes.Buffer{}
	if err := xml.EscapeText(buf, []byte(cmd)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func extractRegistry(path string) (string, string, error) {
	p := strings.Split(path, `\`)
	if len(p) < 2 {
		return "", "", fmt.Errorf("invalid registry path %q", p)
	}
	return p[0], strings.Join(p[1:], `\`), nil
}

func makeRegistryValue(name, typ, value string) RegistryValue {
	return RegistryValue{
		Name:  name,
		Type:  typ,
		Value: value,
	}
}
