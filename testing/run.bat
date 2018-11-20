XCOPY /Y /I /E templates C:\go-msi\templates
go build -o C:\go-msi\go-msi.exe main.go
go run testing\main.go
