package accounts

import (
	"os"
	"path/filepath"
	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
	"github.com/goccy/go-json"

	"core/internal/modules/events"
	sse "core/utils/sse"
	"slices"
)

var (
	AcctDir = filepath.Join(sdkutils.PathConfigDir, "accounts")
)

type Account struct {
	Uname  string   `json:"username"`
	Passwd string   `json:"password"`
	Perms  []string `json:"permissions"`
}

// return the file path of yaml file
func (acct *Account) JsonFile() string {
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
	// TODO: retrieve real permissions
	return []string{sdkapi.AcctPermMaster}
}

// IsMaster returns true if the account is admin
func (acct *Account) IsMaster() bool {
	return slices.Contains(acct.Perms, sdkapi.AcctPermMaster)
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
func (acct *Account) Emit(event string, data []byte) {
	sse.Emit(acct.Username(), event, data)
}

// Save saves the account to yaml file
func (acct *Account) Save() error {
	b, err := json.Marshal(acct)
	if err != nil {
		return err
	}
	return os.WriteFile(acct.JsonFile(), b, sdkutils.PermFile)
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
