# http-proxy
http反向代理

## use
```shell
.\http-proxy.exe -port=8888 -target=http://localhost:8080
.\http-proxy.exe -port 8888 -target tcp://localhost:3306
.\http-proxy.exe -port 8888 -target udp://localhost:8899
```


## build
```shell
go env -w CGO_ENABLED=0 GOOS=windows GOARCH=amd64
go env -w CGO_ENABLED=0 GOOS=linux GOARCH=amd64
go env -w CGO_ENABLED=0 GOOS=darwin3 GOARCH=amd64

```