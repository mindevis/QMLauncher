package auth

import (
	"strings"
)

// CloudAccount represents a QMServer Cloud account
type CloudAccount struct {
	Token    string `json:"token"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// CloudStore holds QMServer Cloud accounts (one primary for now)
type CloudStore struct {
	Accounts []CloudAccount `json:"accounts"`
	Default  string         `json:"default"`
}

// ReadCloudStore returns a snapshot of cloud accounts from the encrypted vault.
func ReadCloudStore() (*CloudStore, error) {
	vaultMu.Lock()
	defer vaultMu.Unlock()
	c := cloudPersisted
	if c.Accounts == nil {
		c.Accounts = []CloudAccount{}
	}
	out := c
	return &out, nil
}

// WriteCloudStore replaces cloud accounts and persists the vault.
func WriteCloudStore(store *CloudStore) error {
	vaultMu.Lock()
	defer vaultMu.Unlock()
	if store == nil {
		cloudPersisted = CloudStore{Accounts: []CloudAccount{}}
	} else {
		cloudPersisted = *store
		if cloudPersisted.Accounts == nil {
			cloudPersisted.Accounts = []CloudAccount{}
		}
	}
	return writeVaultLocked()
}

// AddCloudAccount adds or updates a cloud account by email
func AddCloudAccount(token, email, username string) error {
	vaultMu.Lock()
	defer vaultMu.Unlock()
	email = normalizeEmail(email)
	if username == "" {
		username = emailToUsername(email)
	}
	found := false
	for i := range cloudPersisted.Accounts {
		if normalizeEmail(cloudPersisted.Accounts[i].Email) == email {
			cloudPersisted.Accounts[i].Token = token
			cloudPersisted.Accounts[i].Username = username
			found = true
			break
		}
	}
	if !found {
		cloudPersisted.Accounts = append(cloudPersisted.Accounts, CloudAccount{
			Token:    token,
			Email:    email,
			Username: username,
		})
	}
	if cloudPersisted.Default == "" {
		cloudPersisted.Default = email
	}
	return writeVaultLocked()
}

// GetDefaultCloudAccount returns the default cloud account
func GetDefaultCloudAccount() *CloudAccount {
	vaultMu.Lock()
	defer vaultMu.Unlock()
	if len(cloudPersisted.Accounts) == 0 {
		return nil
	}
	if cloudPersisted.Default != "" {
		for i := range cloudPersisted.Accounts {
			if normalizeEmail(cloudPersisted.Accounts[i].Email) == cloudPersisted.Default {
				c := cloudPersisted.Accounts[i]
				return &c
			}
		}
	}
	c := cloudPersisted.Accounts[0]
	return &c
}

// UpdateDefaultCloudAccountUsername updates the default cloud account's display username (e.g. after syncing local account).
func UpdateDefaultCloudAccountUsername(username string) error {
	if username == "" {
		return nil
	}
	vaultMu.Lock()
	defer vaultMu.Unlock()
	if len(cloudPersisted.Accounts) == 0 {
		return nil
	}
	def := cloudPersisted.Default
	for i := range cloudPersisted.Accounts {
		if def == "" || normalizeEmail(cloudPersisted.Accounts[i].Email) == def {
			cloudPersisted.Accounts[i].Username = strings.TrimSpace(username)
			return writeVaultLocked()
		}
	}
	cloudPersisted.Accounts[0].Username = strings.TrimSpace(username)
	return writeVaultLocked()
}

// RemoveCloudAccount removes a cloud account by email
func RemoveCloudAccount(email string) error {
	vaultMu.Lock()
	defer vaultMu.Unlock()
	email = normalizeEmail(email)
	newAccounts := make([]CloudAccount, 0, len(cloudPersisted.Accounts))
	for _, a := range cloudPersisted.Accounts {
		if normalizeEmail(a.Email) != email {
			newAccounts = append(newAccounts, a)
		}
	}
	cloudPersisted.Accounts = newAccounts
	if normalizeEmail(cloudPersisted.Default) == email {
		cloudPersisted.Default = ""
		if len(cloudPersisted.Accounts) > 0 {
			cloudPersisted.Default = normalizeEmail(cloudPersisted.Accounts[0].Email)
		}
	}
	return writeVaultLocked()
}

func normalizeEmail(e string) string {
	return strings.TrimSpace(strings.ToLower(e))
}

func emailToUsername(email string) string {
	for i, c := range email {
		if c == '@' {
			return email[:i]
		}
	}
	return email
}
