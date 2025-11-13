###

When you encounter `go mod tidy` "sdk/utils" not in your go.mod file, clean the module cache:

```
go clean -modcache
go mod tidy
```
