# Frequently Ask Questions

## How to get client mac address?
To get the client device's MAC address, you can use the [HttpApi.GetClientDevice](../api/http-api.md#getclientdevice) method in your handler:

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, err := api.Http().GetClientDevice(r)
    // handle error
    fmt.Println(clnt.MacAddr()) // print the mac address
}
```

