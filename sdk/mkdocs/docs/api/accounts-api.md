 # IAccountsApi

The `IAccountsApi` allows you to create, modify, remove, and manage admin accounts and permissions. There are two types of admin accounts:

- **normal**: A normal admin account with limited permissions.
- **master**: An admin account that can create, modify, and delete other admin accounts and permissions.

To obtain an instance of the `IAccountsApi`:

```go title="main.go"
import (
    sdkapi "sdk/api"
)
func Init(api sdkapi.IPluginApi) {
    acctAPI := api.Acct()
    fmt.Println(acctAPI) // IAccountsApi
}
```

## 1. IAccountsApi {#accounts-api}

The following are the available methods in `IAccountsApi`:

### Create

It creates a new user account with the given username, password and [permissions](#permissions). It returns an [IAccount](#account-instance) instance and an `error` object.

```go
acctAPI := api.Acct()

username := "admin"
password := "admin"
permissions := []string{sdkapi.AcctPermMaster} // grant "master" permission

acct, err := acctAPI.Create(username, password, permissions)
if err != nil {
    // handle error
}
fmt.Println(acct) // admin account
```

### Find

It finds an user account by the given username. It returns an [IAccount](#account-instance) instance and an `error` object.

```go
acctAPI := api.Acct()
acct, err := acctAPI.Find("admin")
if err != nil {
    // handle error
}
fmt.Println(acct) // IAccount
```

### GetAll

It returns all the normal and master accounts, admin and non-admin. It returns a slice of [IAccount](#account-instance) instance and an `error` object.

```go
acctAPI := api.Acct()
accts, err := acctAPI.GetAll()
if err != nil {
    // handle error
}
fmt.Println(accts) // []IAccount
```

### GetMasterAccts

Get only the master accounts. It returns a slice of [IAccount](#account-instance) instance and an `error` object.

```go
acctAPI := api.Acct()
accts, err := acctAPI.GetMasterAccts()
if err != nil {
    // handle error
}
fmt.Println(accts) // []IAccount
```

### NewPerm

It creates a new permission with the given name and description. It returns an `error` object when an error occured or when the permission already exists.

```go
name := "newperm"
desc := "New permission"
acctAPI := api.Acct()
err := acctAPI.NewPerm(name, desc)
if err != nil {
    // handle error
}
```

### GetAllPerms

It returns all the available permissions, including custom ones from all plugins. The return type is `map[string]string` (name and description pairs of permissions).

```go
acctAPI := api.Acct()
perms := acctAPI.GetAllPerms()
fmt.Println(perms) // map[string]string{"admin": "The admin permission"}
```

### PermDesc

Returns the description of the given permission. It returns a `string`.

```go
desc := api.Acct().PermDesc("newperm")
fmt.Println(desc) // "New permission"
```

---

## 2. IAccount {#account-instance}

`IAccount` represents a system user account. To get an account instance, find an account identified by a username:

```go
acct, err := api.Acct().Find("admin")
if err != nil {
    // handle error
}
fmt.Println(acct) // IAccount
```

Given an user account instance, you can access the following properties and methods:

### Username

It returns the username of the user account.

```go
acct.Username() // "admin"
```

### Permissions

It returns the [permissions](#permissions-sec) of the user account.

```go
acct.Permissions() // []string{"master"}
```

### Update

It updates the user account with the given username, password and [permissions](#permissions). If you don't want to modify the existing account permissions, just pass `nil` value to the permissions parameter. It returns an `error` object.

```go
newUsername := "newadmin"
newPassword := "********"
err := acct.Update(newUsername, newPassword, []string{"master"})
if err != nil {
    // handle error
}
```

### Delete

It deletes the user account. It returns an `error` object. Note: You cannot delete the last user account since it is required for the system to function.

```go
err := acct.Delete()
if err != nil {
    // handle error
}
```

### IsMaster

It returns `true` if the user account has the `master` permission.

```go
acct.IsMaster() // true
```

### HasAllPerms

Returns `true` if the user account has all the given permissions. It can be used to check if an user account has all the required permissions to access a certain part of the system resource or page.

```go
acct, _ := api.Acct().Find("admin")
hasAll := acct.HasAllPerms("master")
fmt.Println(hasAll) // true
```

### HasAnyPerm

It returns `true` if the user account has any of the given permissions. It can be used to check if an user account has any of the required permissions to access a certain part of the system.

```go
acct, _ := api.Acct().Find("admin")
hasAny := acct.HasAnyPerm("master")
fmt.Println(hasAny) // true
```

### Emit

Emit an [event](#events) to the user account. It accepts an event name and data as `[]byte`.
```go
data := []byte(`{"key": "value"}`)
acct.Emit("some_event", data)
```

### Subscribe

You can listen to events emitted to the account using the `Subscribe` method. It returns a channel of `<-chan []byte` that can be marshalled into a json.

```go
ch := acct.Subscribe("some_event")

for b := range ch {
    fmt.Println(string(b))
}
```

### Unsubscribe

You can stop listening to events emitted to the account using the `Unsubscribe` method. It accepts an event name and channel returned by the [Subscribe](#subscribe) method.

```go
ch := acct.Subscribe("some_event")
// Do something with the channel
acct.Unsubscribe("some_event", ch)
```

---

## 3. Permissions {#permissions-sec}

Permissions are used to control the access to various parts of the system. Users without the appropriate permissions will not be able to access the restricted parts of the system.

These are the default permissions that you can assign to an user account. Although you may define your custom permissions using the [AccountsApi.NewPerm](#newperm) method.

| Permission | Description
| --- | --- |
| `master` | Grants full control and access of the system resources. This is aliased by `sdkapi.AcctPermMaster` constant variable. |

---

## 4. Events {#events}

Events are emitted to the user accounts via SSE (Server-Sent Events) in the browser.

You can emit an event to a user account using the [Account.Emit](#emit) method like so:

```go
acct, _ := api.Acct().Find("admin")
acct.Emit("some_event", []byte(`{"key": "value"}`))
```

You can listen to this events in the browser using the [$flare.events](./flare-variable.md#flare-events) like so:

```js
$flare.events.on("some_event", function(data) {
    console.log("An event occured: ", data);
});
```
