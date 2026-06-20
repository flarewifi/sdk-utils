# IMachineApi

The `IMachineApi` provides access to machine-specific information and operations in the Flarewifi system. It allows plugins to access details about the host machine.

To get an instance of `IMachineApi`:

```go
machineAPI := api.Machine()
fmt.Println(machineAPI) // IMachineApi
```

## IMachineApi Methods

The following methods are available in `IMachineApi`:

### GetID

Returns a unique identifier for the machine. This ID is typically used for licensing, machine-specific configurations, or tracking purposes.

```go
machineId := api.Machine().GetID()
fmt.Printf("Machine ID: %s\n", machineId)
```

### IsOnline

Returns whether the machine currently has internet access, as observed by the core's **online monitor** — a background service that periodically probes for internet reachability. This is the same signal that drives `EventInternetUp` / `EventInternetDown` (see [`OnInternetEvent`](./events-api.md#oninternetevent)).

It returns `false` before the monitor's first probe completes — i.e. connectivity is treated as "not known to be up" rather than assumed online.

```go
if api.Machine().IsOnline() {
    // safe to do network-dependent work (cloud sync, remote API calls, etc.)
    syncToCloud()
} else {
    api.Logger().Info("machine is offline; deferring sync")
}
```

> **Prefer events over polling.** If you want to *react* to the machine coming online or going offline, subscribe with [`api.Events().OnInternetEvent(...)`](./events-api.md#oninternetevent) instead of repeatedly calling `IsOnline()`. Use `IsOnline()` for a one-off check at the moment you need it (e.g. just before attempting a network call).

## Usage Examples

### Machine Identification

```go
func getMachineInfo() {
    machineId := api.Machine().GetID()

    // Use machine ID for licensing
    if !isLicensed(machineId) {
        fmt.Println("Machine is not licensed")
        return
    }

    fmt.Printf("Licensed machine: %s\n", machineId)
}
```

### Configuration Per Machine

```go
func loadMachineConfig() map[string]string {
    machineId := api.Machine().GetID()

    // Load configuration specific to this machine
    config := loadConfigForMachine(machineId)

    return config
}
```

### Logging and Debugging

```go
func logWithMachineId(message string) {
    machineId := api.Machine().GetID()
    log.Printf("[Machine: %s] %s", machineId, message)
}

// Usage
logWithMachineId("Plugin initialized successfully")
logWithMachineId("Database connection established")
```

## Security Considerations

- The machine ID should be treated as sensitive information
- Avoid logging machine IDs in production unless necessary for debugging
- Machine IDs can be used for license validation and should be handled securely

## Use Cases

The `IMachineApi` is commonly used for:

- **Licensing**: Validate software licenses tied to specific machines
- **Configuration**: Load machine-specific settings or preferences
- **Logging**: Include machine identification in logs for multi-machine deployments
- **Analytics**: Track usage statistics per machine
- **Security**: Implement machine-based access controls</content>
<parameter name="filePath">/Users/adonesp/Projects/flarehotspot/sdk/mkdocs/docs/api/machine-api.md