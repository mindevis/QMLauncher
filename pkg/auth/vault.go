package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	env "QMLauncher/pkg"
)

const (
	vaultMagic   = "QMLV"
	vaultVersion = byte(1)
)

// credentialsPayload is the plaintext JSON stored inside the encrypted vault.
type credentialsPayload struct {
	Version   int                `json:"v"`
	Microsoft AuthStore          `json:"microsoft"`
	Local     LocalAccountsStore `json:"local"`
	Cloud     CloudStore         `json:"cloud"`
}

var (
	vaultMu sync.Mutex

	// cloudPersisted holds QMServer Cloud accounts (also serialized inside the vault).
	cloudPersisted CloudStore
)

// LoadCredentials loads Microsoft, offline and cloud accounts from the encrypted vault.
// On first run after upgrade, migrates account.json, local_accounts.json and cloud_accounts.json
// into the vault and removes those files (and .launcher_history if present).
func LoadCredentials() error {
	vaultMu.Lock()
	defer vaultMu.Unlock()

	vaultPath := env.CredentialsVaultPath
	_, statErr := os.Stat(vaultPath)
	if statErr == nil {
		if err := readVaultLocked(vaultPath); err != nil {
			return err
		}
		normalizeLoadedLocalAccountsLocked()
		return nil
	}
	if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("stat vault: %w", statErr)
	}

	// Missing vault — try legacy plaintext migration
	migrated := migrateLegacyPlaintextCredentialsLocked()
	normalizeLoadedLocalAccountsLocked()
	if migrated {
		if err := writeVaultLocked(); err != nil {
			return err
		}
		removeLegacyArtifactsLocked()
	}
	if LocalStore.Accounts == nil {
		LocalStore.Accounts = []LocalAccount{}
	}
	if cloudPersisted.Accounts == nil {
		cloudPersisted.Accounts = []CloudAccount{}
	}
	return nil
}

func normalizeLoadedLocalAccountsLocked() {
	needsSave := false
	for i := range LocalStore.Accounts {
		if LocalStore.Accounts[i].UUID == "" {
			LocalStore.Accounts[i].UUID = uuidForSkinModel(LocalStore.Accounts[i].SkinModel)
			if LocalStore.Accounts[i].SkinModel == "" {
				LocalStore.Accounts[i].SkinModel = SkinModelSteve
			}
			needsSave = true
		}
	}
	if needsSave {
		_ = writeVaultLocked()
	}
}

func migrateLegacyPlaintextCredentialsLocked() bool {
	any := false
	accountPath := filepath.Join(env.RootDir, "account.json")
	localPath := filepath.Join(env.RootDir, "local_accounts.json")
	cloudPath := filepath.Join(env.RootDir, "cloud_accounts.json")

	if data, err := os.ReadFile(accountPath); err == nil && len(data) > 0 {
		var s AuthStore
		if json.Unmarshal(data, &s) == nil {
			Store = s
			any = true
		}
	}

	if data, err := os.ReadFile(localPath); err == nil && len(data) > 0 {
		var s LocalAccountsStore
		if json.Unmarshal(data, &s) == nil {
			LocalStore = s
			any = true
		}
	}

	if data, err := os.ReadFile(cloudPath); err == nil && len(data) > 0 {
		var s CloudStore
		if json.Unmarshal(data, &s) == nil {
			cloudPersisted = s
			any = true
		}
	}

	if cloudPersisted.Accounts == nil {
		cloudPersisted.Accounts = []CloudAccount{}
	}
	if LocalStore.Accounts == nil {
		LocalStore.Accounts = []LocalAccount{}
	}

	return any
}

func removeLegacyArtifactsLocked() {
	_ = os.Remove(filepath.Join(env.RootDir, "account.json"))
	_ = os.Remove(filepath.Join(env.RootDir, "local_accounts.json"))
	_ = os.Remove(filepath.Join(env.RootDir, "cloud_accounts.json"))
	_ = os.Remove(filepath.Join(env.RootDir, ".launcher_history"))
}

func deriveVaultKey(salt []byte) ([32]byte, error) {
	if len(salt) != 16 {
		return [32]byte{}, errors.New("invalid salt length")
	}
	h := sha256.New()
	h.Write([]byte("QMLauncher/credentials.v1\x00"))
	h.Write(salt)
	h.Write([]byte(env.RootDir))
	h.Write([]byte{0})
	host, err := os.Hostname()
	if err == nil && host != "" {
		h.Write([]byte(host))
	}
	var key [32]byte
	copy(key[:], h.Sum(nil))
	return key, nil
}

func readVaultLocked(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(raw) < len(vaultMagic)+1+16+12 {
		return errors.New("vault file too short")
	}
	if string(raw[:len(vaultMagic)]) != vaultMagic {
		return errors.New("invalid vault magic")
	}
	if raw[len(vaultMagic)] != vaultVersion {
		return fmt.Errorf("unsupported vault version %d", raw[len(vaultMagic)])
	}
	salt := raw[len(vaultMagic)+1 : len(vaultMagic)+1+16]
	nonce := raw[len(vaultMagic)+1+16 : len(vaultMagic)+1+16+12]
	ct := raw[len(vaultMagic)+1+16+12:]

	key, err := deriveVaultKey(salt)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return fmt.Errorf("decrypt vault: %w", err)
	}

	var payload credentialsPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return fmt.Errorf("parse vault json: %w", err)
	}
	Store = payload.Microsoft
	LocalStore = payload.Local
	if LocalStore.Accounts == nil {
		LocalStore.Accounts = []LocalAccount{}
	}
	cloudPersisted = payload.Cloud
	if cloudPersisted.Accounts == nil {
		cloudPersisted.Accounts = []CloudAccount{}
	}
	return nil
}

func writeVaultLocked() error {
	if err := os.MkdirAll(env.RootDir, 0755); err != nil {
		return err
	}

	payload := credentialsPayload{
		Version:   1,
		Microsoft: Store,
		Local:     LocalStore,
		Cloud:     cloudPersisted,
	}
	plain, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}
	key, err := deriveVaultKey(salt)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	ct := gcm.Seal(nil, nonce, plain, nil)

	out := make([]byte, 0, len(vaultMagic)+1+len(salt)+len(nonce)+len(ct))
	out = append(out, []byte(vaultMagic)...)
	out = append(out, vaultVersion)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ct...)

	tmp := env.CredentialsVaultPath + ".tmp"
	if err := os.WriteFile(tmp, out, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, env.CredentialsVaultPath)
}

func persistVault() error {
	vaultMu.Lock()
	defer vaultMu.Unlock()
	return writeVaultLocked()
}
