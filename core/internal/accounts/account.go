package accounts

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-json"

	"core/internal/utils/events"
	sse "core/internal/utils/sse"

	sdkfs "github.com/flarehotspot/go-utils/fs"
	paths "github.com/flarehotspot/go-utils/paths"
)

var (
	AcctDir = filepath.Join(paths.ConfigDir, "accounts")
)

type Account struct {
	Uname  string   `json:"username"`
	Passwd string   `json:"password"`
	Perms  []string `json:"permissions"`
}

// return the file path of yaml file
func (acct *Account) YamlFile() string {
	return FilepathForUser(acct.Uname)
}

// Username returns the username of the account
func (acct *Account) Username() string {
	return acct.Uname
}

// Auth returns true if the password is correct
func (acct *Account) Auth(pw string) bool {
	return acct.Passwd == pw
}

// Permissions returns the permissions of the account
func (acct *Account) Permissions() []string {
	return acct.Perms
}

// IsAdmin returns true if the account is admin
func (acct *Account) IsAdmin() bool {
	for _, p := range acct.Perms {
		if p == PermAdmin {
			return true
		}
	}
	return false
}

// HasAllPerms
func (acct *Account) HasAllPerms(perms ...string) bool {
	return HasAllPerms(acct, perms...)
}

// HasAnyPerm
func (acct *Account) HasAnyPerm(perms ...string) bool {
	return HasAnyPerm(acct, perms...)
}

// AddSocket adds a sse socket to the account
func (acct *Account) AddSocket(s *sse.SseSocket) {
	sse.AddSocket(acct.Username(), s)
}

// Emit emits an event to the account that will propage to the browser
func (acct *Account) Emit(event string, data interface{}) {
	sse.Emit(acct.Username(), event, data)
}

// Save saves the account to yaml file
func (acct *Account) Save() error {
	b, err := json.Marshal(acct)
	if err != nil {
		return err
	}
	return os.WriteFile(acct.YamlFile(), b, sdkfs.PermFile)
}

// Update updates the account with new username, password and permissions
func (acct *Account) Update(uname string, pass string, perms []string) error {
	_, err := Update(acct.Uname, uname, pass, perms)
	if err != nil {
		return err
	}

	acct.Uname = uname
	acct.Passwd = pass
	acct.Perms = perms
	return nil
}

// Delete deletes the account
func (acct *Account) Delete() error {
	return Delete(acct.Uname)
}

func (acct *Account) Subscribe(event string) <-chan []byte {
	channel := acct.GetChannel()
	return events.Subscribe(channel)
}

func (acct *Account) Unsubscribe(event string, ch <-chan []byte) {
	channel := acct.GetChannel()
	events.Unsubscribe(channel, ch)
}

func (acct *Account) GetChannel() string {
	return "account:" + acct.Uname
}
