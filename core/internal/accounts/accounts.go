package accounts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/goccy/go-json"

	accounts "sdk/api/accounts"

	fs "github.com/flarehotspot/go-utils/fs"
	sdkfs "github.com/flarehotspot/go-utils/fs"
	paths "github.com/flarehotspot/go-utils/paths"
)

const (
	AdminUsername = "admin"
	AdminPassword = "admin"
	PermAdmin     = "admin"
)

var (
	perms        sync.Map
	DefaultPerms = []string{PermAdmin}
	ErrNoAccount = errors.New("Account does not exist")
)

func init() {
	perms.Store(PermAdmin, "Manage Users")
}

func DefaultAdminAcct() Account {
	f := filepath.Join(paths.DefaultsDir, "admin.json")

	perms := []string{}
	for _, p := range DefaultPerms {
		perms = append(perms, p)
	}

	defAcct := Account{
		Uname:  AdminUsername,
		Passwd: AdminPassword,
		Perms:  perms,
	}

	var acct Account
	if err := sdkfs.ReadJson(f, &acct); err != nil {
		return defAcct
	}

	return acct
}

func EnsureAdminAcct() error {
	f := FilepathForUser(AdminUsername)
	if !fs.Exists(f) {
		acct := DefaultAdminAcct()
		content, err := json.Marshal(acct)
		if err != nil {
			return err
		}
		err = os.WriteFile(f, content, sdkfs.PermFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func All() (accounts []*Account, err error) {
	files := []string{}
	if err := fs.LsFiles(AcctDir, &files, false); err != nil {
		return nil, err
	}

	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}

		var acct Account
		err = json.Unmarshal(b, &acct)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, &acct)
	}

	return accounts, err
}

func AllAdmins() ([]*Account, error) {
	accts, err := All()
	if err != nil {
		return nil, err
	}

	admins := []*Account{}
	for _, acct := range accts {
		if acct.IsAdmin() {
			admins = append(admins, acct)
		}
	}

	return admins, nil
}

func Find(username string) (*Account, error) {
	var acct Account
	f := FilepathForUser(username)
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, ErrNoAccount
	}
	err = json.Unmarshal(b, &acct)
	if err != nil {
		return &acct, ErrNoAccount
	}
	return &acct, nil
}

func Create(uname string, passwd string, perms []string) (*Account, error) {
	acct := Account{
		Uname:  uname,
		Passwd: passwd,
		Perms:  perms,
	}

	b, err := json.Marshal(&acct)
	if err != nil {
		return nil, err
	}

	f := FilepathForUser(uname)
	if fs.Exists(f) {
		return nil, fmt.Errorf("Account with username \"%s\" already exists", uname)
	}

	err = os.WriteFile(f, b, sdkfs.PermFile)
	if err != nil {
		return nil, err
	}

	return &acct, nil
}

func Update(prevName string, newName string, pass string, perms []string) (*Account, error) {
	prevAcct, err := Find(prevName)
	if err != nil {
		return nil, err
	}

	if prevAcct.Uname == AdminUsername && newName != AdminUsername {
		return nil, errors.New("Renaming the super admin account is not allowed.")
	}

	if pass == "" {
		pass = prevAcct.Passwd
	}

	if len(perms) == 0 {
		perms = prevAcct.Perms
	}

	acct := Account{
		Uname:  newName,
		Passwd: pass,
		Perms:  perms,
	}

	if acct.Uname == AdminUsername && !HasAllPerms(&acct, PermAdmin) {
		return nil, errors.New("Super admin account must have manage users permission.")
	}

	f := FilepathForUser(newName)

	err = sdkfs.WriteJson(f, acct)
	if err != nil {
		return nil, err
	}

	if prevName != newName {
		f = FilepathForUser(prevName)
		err = os.Remove(f)
		return &acct, err
	}

	return &acct, nil
}

func Delete(uname string) error {
	if uname == AdminUsername {
		return fmt.Errorf("Deleting the super admin account is not allowed.")
	}

	files := []string{}
	if err := fs.LsFiles(AcctDir, &files, false); err != nil {
		return err
	}

	acct, err := Find(uname)
	if err != nil {
		return err
	}

	if len(files) < 2 && acct.Uname == uname {
		return errors.New("Can't delete last super admin user.")
	}

	return os.Remove(FilepathForUser(uname))
}

func FilepathForUser(uname string) string {
	return filepath.Join(AcctDir, uname+".json")
}

// Permissions returns all permissions from perms.SyncMap as map[string]string
func Permissions() map[string]string {
	permsMap := map[string]string{}
	perms.Range(func(key, value interface{}) bool {
		permsMap[key.(string)] = value.(string)
		return true
	})
	return permsMap
}

// PermDesc returns description string of permission name
func PermDesc(perm string) string {
	desc, ok := perms.Load(perm)
	if !ok {
		return ""
	}
	return desc.(string)
}

// Check if account has all permissions
func HasAllPerms(acct accounts.IAccount, perms ...string) bool {
	count := 0
	for _, perm := range perms {
		for _, acctPerm := range acct.Permissions() {
			if perm == acctPerm {
				count++
			}
		}
	}

	return count == len(perms)
}

// Check if account has any of the permissions
func HasAnyPerm(acct accounts.IAccount, perms ...string) bool {
	for _, perm := range perms {
		for _, acctPerm := range acct.Permissions() {
			if perm == acctPerm {
				return true
			}
		}
	}
	return false
}

// Add new permission to perms sync.Map with name and description params
func NewPerm(name string, description string) error {
	_, ok := perms.Load(name)
	if ok {
		return errors.New("Permission with name \"" + name + "\" already exists")
	}

	perms.Store(name, description)
	return nil
}
