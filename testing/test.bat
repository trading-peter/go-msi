
cd C:\\gopath\\src\\github.com\\mat007\\go-msi\\testing\\hello

C:\\go-msi\\go-msi.exe make --msi hello.msi --version 0.0.1 --arch amd64
msiexec /i hello.msi /q

mkdir wixtemplates
C:\\go-msi\\go-msi.exe generate-templates --out wixtemplates --version 0.0.1
