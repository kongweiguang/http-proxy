# http-proxy
http反向代理

## use
```shell
.\http-proxy.exe -port=8888 -target=http://localhost:8003
```


## build
```shell
go env -w CGO_ENABLED=0 GOOS=windows GOARCH=amd64
go env -w CGO_ENABLED=0 GOOS=linux GOARCH=amd64
go env -w CGO_ENABLED=0 GOOS=darwin3 GOARCH=amd64

```