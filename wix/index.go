package wix

import (
	"path/filepath"
	"strings"

	"github.com/observiq/go-msi/manifest"
)

var eol = "\r\n"

// GenerateCmd generates required command lines to produce an msi package,
func GenerateCmd(wixFile *manifest.WixManifest, templates []string, msiOutFile, arch, path, lang, langFile string) string {
	cmd := ""

	cmd += filepath.Join(path, "candle")
	if arch != "" {
		if arch == "386" {
			arch = "x86"
		} else if arch == "amd64" {
			arch = "x64"
		}
		cmd += " -arch " + arch
	}
	cmd += " -ext WixUIExtension -ext WixUtilExtension"
	for _, tpl := range templates {
		cmd += " " + filepath.Base(tpl)
	}
	cmd += eol
	cmd += filepath.Join(path, "light") + " -ext WixUIExtension -ext WixUtilExtension -sacl -spdb "
	cmd += " -out " + msiOutFile
	if lang != "" {
		cmd += " -cultures:" + lang
		cmd += " -loc " + langFile
	}
	for _, tpl := range templates {
		cmd += " " + strings.Replace(filepath.Base(tpl), ".wxs", ".wixobj", -1)
	}
	cmd += eol

	return cmd
}
