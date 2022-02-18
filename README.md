# go-msi

Forked from https://github.com/mh-cbon/go-msi

[![Appveyor Status](https://ci.appveyor.com/api/projects/status/github/mat007/go-msi?branch=master&svg=true)](https://ci.appveyor.com/project/mat007/go-msi)

go-msi leverages the [WiX Toolset](http://wixtoolset.org) to create a Windows MSI package from a JSON description of the product to install.

A demo program can be seen [here](https://github.com/observiq/go-msi/tree/master/testing/hello).

## Install

Download from the [release page](https://github.com/observiq/go-msi/releases) or use `go get`:
```sh
go get github.com/observiq/go-msi
```

## Usage

### Requirements

go-msi needs [WiX Toolset](http://wixtoolset.org/) 3.10 or later

### Workflow

- Create a `wix.json` file like [this one](https://github.com/observiq/go-msi/blob/master/testing/hello/wix.json)
- Leave the `upgrade-code` empty or remove it all together
- Assign a fresh `upgrade-code` with `go-msi set-guid`, this must be done only once
- Run `go-msi make --msi your_program.msi --version 0.0.1`

### configuration file

The `wix.json` file describes the packaging rules for bundling the product files into the MSI package.

Check the demo [wix.json](https://github.com/observiq/go-msi/blob/master/testing/hello/wix.json) file.

### License file

The license file must be in RTF and encoded with the `Windows1252` charset.

## Customization

The WiX template files (in the [templates](templates) folder) can be modified to personnalize the behaviour of the MSI package.

## Command line

###### $ go-msi -h
```
NAME:
   go-msi - Easy msi pakage for Go

USAGE:
   go-msi <cmd> <options>

VERSION:
   0.0.0

COMMANDS:
     check-json          Check the JSON wix manifest
     check-env           Provide a report about your environment setup
     set-files           Adds or removes files from your wix manifest
     set-guid            Sets appropriate guids in your wix manifest
     generate-templates  Generate wix templates
     to-windows          Write Windows1252 encoded file
     to-rtf              Write RTF formatted file
     gen-wix-cmd         Generate a batch file of Wix commands to run
     run-wix-cmd         Run the batch file of Wix commands
     make                All-in-one command to make MSI files
     choco               Generate a chocolatey package of your msi files
     help, h             Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

###### $ go-msi check-env -h
```
NAME:
   go-msi check-env - Provide a report about your environment setup

USAGE:
   go-msi check-env [arguments...]
```

###### $ go-msi check-json -h
```
NAME:
   go-msi check-json - Check the JSON wix manifest

USAGE:
   go-msi check-json [command options] [arguments...]

OPTIONS:
   --path value, -p value  Path to the wix manifest file (default: "wix.json")
```

###### $ go-msi set-files -h
```
NAME:
   go-msi.exe set-files - Adds or removes files from your wix manifest

USAGE:
   go-msi.exe set-files [command options] [arguments...]

OPTIONS:
   --path value, -p value      Path to the wix manifest file (default: "wix.json")
   --includes value, -i value  Files to include, use of * is permitted
   --excludes value, -e value  Files to exclude, use of * is permitted
   --test, -t                  Test mode, does not modify the wix manifest file but exits with an error instead
```

###### $ go-msi set-guid -h
```
NAME:
   go-msi set-guid - Sets appropriate guids in your wix manifest

USAGE:
   go-msi set-guid [command options] [arguments...]

OPTIONS:
   --path value, -p value  Path to the wix manifest file (default: "wix.json")
   --force, -f             Force update the guids
```

###### $ go-msi make -h
```
NAME:
   go-msi make - All-in-one command to make MSI files

USAGE:
   go-msi make [command options] [arguments...]

OPTIONS:
   --path value, -p value     Path to the wix manifest file (default: "wix.json")
   --src value, -s value      Directory path to the wix templates files (default: "/home/mat007/gow/bin/templates")
   --out value, -o value      Directory path to the generated wix cmd file (default: "/tmp/go-msi645264968")
   --arch value, -a value     A target architecture, amd64 or 386 (ia64 is not handled)
   --msi value, -m value      Path to write resulting msi file to
   --version value            The version of your program
   --license value, -l value  Path to the license file
   --keep, -k                 Keep output directory containing build files (useful for debug)
```

###### $ go-msi choco -h
```
NAME:
   go-msi choco - Generate a chocolatey package of your msi files

USAGE:
   go-msi choco [command options] [arguments...]

OPTIONS:
   --path value, -p value           Path to the wix manifest file (default: "wix.json")
   --src value, -s value            Directory path to the wix templates files (default: "/home/mat007/gow/bin/templates/choco")
   --version value                  The version of your program
   --out value, -o value            Directory path to the generated chocolatey build file (default: "/tmp/go-msi697894350")
   --input value, -i value          Path to the msi file to package into the chocolatey package
   --changelog-cmd value, -c value  A command to generate the content of the changlog in the package
   --keep, -k                       Keep output directory containing build files (useful for debug)
```

###### $ go-msi generate-templates -h
```
NAME:
   go-msi generate-templates - Generate wix templates

USAGE:
   go-msi generate-templates [command options] [arguments...]

OPTIONS:
   --path value, -p value     Path to the wix manifest file (default: "wix.json")
   --src value, -s value      Directory path to the wix templates files (default: "/home/mat007/gow/bin/templates")
   --out value, -o value      Directory path to the generated wix templates files (default: "/tmp/go-msi522345138")
   --version value            The version of your program
   --license value, -l value  Path to the license file
```

###### $ go-msi to-windows -h
```
NAME:
   go-msi to-windows - Write Windows1252 encoded file

USAGE:
   go-msi to-windows [command options] [arguments...]

OPTIONS:
   --src value, -s value  Path to an UTF-8 encoded file
   --out value, -o value  Path to the ANSI generated file
```

###### $ go-msi to-rtf -h
```
NAME:
   go-msi to-rtf - Write RTF formatted file

USAGE:
   go-msi to-rtf [command options] [arguments...]

OPTIONS:
   --src value, -s value  Path to a text file
   --out value, -o value  Path to the RTF generated file
   --reencode, -e         Also re encode UTF-8 to Windows1252 charset
```

###### $ go-msi gen-wix-cmd -h
```
NAME:
   go-msi gen-wix-cmd - Generate a batch file of Wix commands to run

USAGE:
   go-msi gen-wix-cmd [command options] [arguments...]

OPTIONS:
   --path value, -p value  Path to the wix manifest file (default: "wix.json")
   --src value, -s value   Directory path to the wix templates files (default: "/home/mat007/gow/bin/templates")
   --out value, -o value   Directory path to the generated wix cmd file (default: "/tmp/go-msi844736928")
   --arch value, -a value  A target architecture, amd64 or 386 (ia64 is not handled)
   --msi value, -m value   Path to write resulting msi file to
```

###### $ go-msi run-wix-cmd -h
```
NAME:
   go-msi run-wix-cmd - Run the batch file of Wix commands

USAGE:
   go-msi run-wix-cmd [command options] [arguments...]

OPTIONS:
   --out value, -o value  Directory path to the generated wix cmd file (default: "/tmp/go-msi773158361")
```

# History

[CHANGELOG](CHANGELOG.md)

# Credits

Thanks to `mh-cbon` for providing https://github.com/mh-cbon/go-msi from which this project has been forked.
