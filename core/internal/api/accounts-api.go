package api

import (
	"core/internal/accounts"
	sdkapi "sdk/api"
)

func NewAcctApi(api *PluginApi) {
	acctApi := &AccountsApi{api}
	api.AcctAPI = acctApi
}

type AccountsApi struct {
	api *PluginApi
}

func (self *AccountsApi) Create(uname string, pass string, perms []string) (sdkapi.IAccount, error) {
	return accounts.Create(uname, pass, perms)
}

func (self *AccountsApi) Find(username string) (sdkapi.IAccount, error) {
	return accounts.Find(username)
}

func (self *AccountsApi) Update(oldname string, uname string, pass string, perms []string) (sdkapi.IAccount, error) {
	return accounts.Update(oldname, uname, pass, perms)
}

func (self *AccountsApi) Delete(uname string) error {
	return accounts.Delete(uname)
}

func (self *AccountsApi) GetAll() ([]sdkapi.IAccount, error) {
	accts, err := accounts.All()
	if err != nil {
		return nil, err
	}

	accounts := []sdkapi.IAccount{}
	for _, a := range accts {
		accounts = append(accounts, a)
	}

	return accounts, nil
}

func (self *AccountsApi) GetMasterAccts() ([]sdkapi.IAccount, error) {
	accts, err := accounts.All()
	if err != nil {
		return nil, err
	}

	admins := []sdkapi.IAccount{}
	for _, acct := range accts {
		if acct.IsMaster() {
			admins = append(admins, acct)
		}
	}

	return admins, nil
}

func (self *AccountsApi) NewPerm(name string, desc string) error {
	return accounts.NewPerm(name, desc)
}

func (self *AccountsApi) GetAllPerms() map[string]string {
	return accounts.Permissions()
}

func (self *AccountsApi) PermDesc(name string) (desc string) {
	return accounts.PermDesc(name)
}

func (self *AccountsApi) HasAllPerms(acct sdkapi.IAccount, perms ...string) bool {
	return accounts.HasAllPerms(acct, perms...)
}

func (self *AccountsApi) HasAnyPerm(acct sdkapi.IAccount, perms ...string) bool {
	return accounts.HasAnyPerm(acct, perms...)
}
