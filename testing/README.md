# e2e testing - go-msi

End-to-end testing with Docker on Windows containers.

From project root:
```bat
docker build -f testing/Dockerfile -t go-msi-testing:latest testing &&
  docker run --rm -it -v C:/dev/src/github.com/observiq/go-msi:C:/gopath/src/github.com/observiq/go-msi go-msi-testing:latest
```

then from within the container:
```bat
XCOPY /Y /I /E templates C:\go-msi\templates
go build -o C:\go-msi\go-msi.exe main.go
go run testing\main.go
```
