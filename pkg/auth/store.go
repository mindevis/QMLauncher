package auth

import (
	"net/url"
	"time"

	"github.com/google/uuid"
)

// Store is the global authentication store.
var Store AuthStore

// LocalStore is the global local accounts store.
var LocalStore LocalAccountsStore

type msaAuthStore struct {
	AccessToken  string    `json:"access_token"`
	Expires      time.Time `json:"expires"`
	RefreshToken string    `json:"refresh_token"`
}

func (store *msaAuthStore) isValid() bool {
	return store.AccessToken != "" && store.Expires.After(time.Now())
}
func (store *msaAuthStore) refresh() error {
	resp, err := authenticateMSA(url.Values{
		"client_id":     {ClientID},
		"scope":         {scope},
		"grant_type":    {"refresh_token"},
		"refresh_token": {Store.MSA.RefreshToken},
	})
	if err != nil {
		return err
	}
	store.write(resp)
	return nil
}
func (store *msaAuthStore) write(resp msaResponse) {
	store.AccessToken = resp.AccessToken
	store.Expires = time.Now().Add(time.Second * time.Duration(resp.ExpiresIn))
	store.RefreshToken = resp.RefreshToken
}

type xblAuthStore struct {
	Userhash string    `json:"uhs"`
	Token    string    `json:"token"`
	Expires  time.Time `json:"expires"`
}

func (store *xblAuthStore) isValid() bool {
	return store.Token != "" && store.Userhash != "" && store.Expires.After(time.Now())
}
func (store *xblAuthStore) refresh() error {
	resp, err := authenticateXBL(Store.MSA.AccessToken)
	if err != nil {
		return err
	}
	store.write(resp)
	return nil
}
func (store *xblAuthStore) write(resp xblResponse) {
	store.Userhash = resp.DisplayClaims.Xui[0].Uhs
	store.Token = resp.Token
	store.Expires = resp.NotAfter
}

type xstsAuthStore struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

func (store *xstsAuthStore) isValid() bool {
	return store.Token != "" && store.Expires.After(time.Now())
}
func (store *xstsAuthStore) refresh() error {
	resp, err := authenticateXSTS(Store.XBL.Token)
	if err != nil {
		return err
	}
	store.write(resp)
	return nil
}
func (store *xstsAuthStore) write(resp xstsResponse) {
	store.Token = resp.Token
	store.Expires = resp.NotAfter
}

type minecraftAuthStore struct {
	AccessToken string    `json:"access_token"`
	Expires     time.Time `json:"expires"`
	Username    string    `json:"name"`
	UUID        string    `json:"id"`
}

func (store *minecraftAuthStore) isValid() bool {
	return store.AccessToken != "" && store.Expires.After(time.Now())
}
func (store *minecraftAuthStore) refresh() error {
	resp, profile, err := authenticateMinecraft(Store.XSTS.Token, Store.XBL.Userhash)
	if err != nil {
		return err
	}
	store.write(resp, profile)
	return nil
}
func (store *minecraftAuthStore) write(resp minecraftResponse, profile minecraftProfile) {
	store.AccessToken = resp.AccessToken
	store.Expires = time.Now().Add(time.Second * time.Duration(resp.ExpiresIn))
	store.Username = profile.Name
	store.UUID = profile.ID
}

// SkinModel is the default Minecraft skin model: "steve" (male/classic) or "alex" (female/slim)
const SkinModelSteve = "steve"
const SkinModelAlex = "alex"

// LocalAccount represents a local offline account
type LocalAccount struct {
	Name      string `json:"name"`
	Type      string `json:"type"`                 // Always "local"
	UUID      string `json:"uuid"`                 // Unique UUID for each account (Minecraft offline mode)
	SkinModel string `json:"skin_model,omitempty"` // "steve" or "alex" - determines default skin
}

// LocalAccountsStore manages local accounts
type LocalAccountsStore struct {
	Accounts       []LocalAccount `json:"accounts"`
	DefaultAccount string         `json:"default_account"` // Name of default account
}

// An AuthStore is an authentication store which stores necessary information to log in.
type AuthStore struct {
	MSA       msaAuthStore       `json:"msa"`
	XBL       xblAuthStore       `json:"xbl"`
	XSTS      xstsAuthStore      `json:"xsts"`
	Minecraft minecraftAuthStore `json:"minecraft"`
}

// WriteToCache persists Microsoft session data into the encrypted credentials vault.
func (store *AuthStore) WriteToCache() error {
	Store = *store
	return persistVault()
}

// Clear clears the Microsoft session and persists the vault.
func (store *AuthStore) Clear() error {
	*store = AuthStore{}
	Store = *store
	return persistVault()
}

// ReadFromCache loads all credentials from the vault (same as LoadCredentials).
func ReadFromCache() error {
	return LoadCredentials()
}

// WriteLocalAccountsToCache writes offline accounts into the encrypted vault.
func (store *LocalAccountsStore) WriteToCache() error {
	LocalStore = *store
	return persistVault()
}

// ReadLocalAccountsFromCache loads all credentials from the vault (same as LoadCredentials).
func ReadLocalAccountsFromCache() error {
	return LoadCredentials()
}

// uuidHashCode replicates Java's UUID.hashCode() - used for Minecraft Steve/Alex skin selection.
// Minecraft: (hashCode & 1) == 0 → Steve (classic), == 1 → Alex (slim)
func uuidHashCode(u uuid.UUID) int {
	mostSig := int64(u[0])<<56 | int64(u[1])<<48 | int64(u[2])<<40 | int64(u[3])<<32 |
		int64(u[4])<<24 | int64(u[5])<<16 | int64(u[6])<<8 | int64(u[7])
	leastSig := int64(u[8])<<56 | int64(u[9])<<48 | int64(u[10])<<40 | int64(u[11])<<32 |
		int64(u[12])<<24 | int64(u[13])<<16 | int64(u[14])<<8 | int64(u[15])
	hilo := mostSig ^ leastSig
	return int(hilo>>32) ^ int(hilo)
}

// uuidForSkinModel generates a UUID that produces the desired default skin in Minecraft.
// skinModel: "alex" (slim/female) or "steve" (classic/male, default)
func uuidForSkinModel(skinModel string) string {
	wantAlex := skinModel == SkinModelAlex
	for i := 0; i < 100; i++ {
		id := uuid.New()
		hash := uuidHashCode(id)
		isAlex := (hash & 1) == 1
		if isAlex == wantAlex {
			return id.String()
		}
	}
	return uuid.New().String()
}

// AddLocalAccount adds a new local account with a unique UUID.
// skinModel: "steve" (male/classic) or "alex" (female/slim) - determines default skin in game
func (store *LocalAccountsStore) AddLocalAccount(name string, skinModel string) {
	if skinModel != SkinModelAlex {
		skinModel = SkinModelSteve
	}
	account := LocalAccount{
		Name:      name,
		Type:      "local",
		UUID:      uuidForSkinModel(skinModel),
		SkinModel: skinModel,
	}
	store.Accounts = append(store.Accounts, account)

	// If this is the first account, make it default
	if len(store.Accounts) == 1 {
		store.DefaultAccount = name
	}

	store.WriteToCache()
}

// RemoveLocalAccount removes a local account by name.
// Clears DefaultAccount if the removed account was the default.
func (store *LocalAccountsStore) RemoveLocalAccount(name string) {
	for i, account := range store.Accounts {
		if account.Name == name {
			store.Accounts = append(store.Accounts[:i], store.Accounts[i+1:]...)
			if store.DefaultAccount == name {
				store.DefaultAccount = ""
			}
			store.WriteToCache()
			return
		}
	}
}

// GetLocalAccountNames returns a list of local account names.
func (store *LocalAccountsStore) GetLocalAccountNames() []string {
	names := make([]string, len(store.Accounts))
	for i, account := range store.Accounts {
		names[i] = account.Name
	}
	return names
}

// SetDefaultAccount sets the default account by name.
func (store *LocalAccountsStore) SetDefaultAccount(name string) {
	// Check if account exists
	for _, account := range store.Accounts {
		if account.Name == name {
			store.DefaultAccount = name
			store.WriteToCache()
			return
		}
	}
}

// GetDefaultAccount returns the name of the default account.
func (store *LocalAccountsStore) GetDefaultAccount() string {
	return store.DefaultAccount
}

// ClearDefaultIfInvalid clears DefaultAccount if it doesn't exist in Accounts (e.g. after sync to cloud).
func (store *LocalAccountsStore) ClearDefaultIfInvalid() {
	if store.DefaultAccount == "" {
		return
	}
	for _, acc := range store.Accounts {
		if acc.Name == store.DefaultAccount {
			return
		}
	}
	store.DefaultAccount = ""
	store.WriteToCache()
}

// GetLocalAccountNames returns a list of local account names from global store.
func GetLocalAccountNames() []string {
	return LocalStore.GetLocalAccountNames()
}

// GetLocalAccountByName returns a local account by name, or nil if not found.
func GetLocalAccountByName(name string) *LocalAccount {
	for i := range LocalStore.Accounts {
		if LocalStore.Accounts[i].Name == name {
			return &LocalStore.Accounts[i]
		}
	}
	return nil
}

// EnsureLocalAccountUUID ensures the account has a UUID (for sync to cloud). Returns the UUID.
// If missing, generates one preserving skin model and saves to store.
func EnsureLocalAccountUUID(acc *LocalAccount) string {
	if acc == nil {
		return ""
	}
	if acc.UUID != "" {
		return acc.UUID
	}
	acc.UUID = uuidForSkinModel(acc.SkinModel)
	if acc.SkinModel == "" {
		acc.SkinModel = SkinModelSteve
	}
	// Update in store
	for i := range LocalStore.Accounts {
		if LocalStore.Accounts[i].Name == acc.Name {
			LocalStore.Accounts[i] = *acc
			LocalStore.WriteToCache()
			break
		}
	}
	return acc.UUID
}

// AddLocalAccount adds a new local account to global store.
// skinModel: "steve" or "alex"
func AddLocalAccount(name string, skinModel string) {
	LocalStore.AddLocalAccount(name, skinModel)
}
