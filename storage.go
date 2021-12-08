package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
)

type AppStorage struct {
	KnownAccounts []Account `json:"known_accounts"`
}

type Account struct {
	Id       string `json:"id"`
	Nickname string `json:"nickname"`
}

var (
	_storageFile string
	_storage     AppStorage
	_storageLock sync.Mutex
)

func storageInit() {
	_storageFile = filepath.Join(_internalDir, "storage.json")
	loadStorage()
}

func loadStorage() {
	_storageLock.Lock()
	defer _storageLock.Unlock()
	encoded, err := os.ReadFile(_storageFile)
	if err != nil {
		log.Errorf("error loading storage.json: %s", err)
		return
	}
	if err := json.Unmarshal(encoded, &_storage); err != nil {
		log.Errorf("error parsing storage.json: %s", err)
		return
	}
}

func persistStorage() {
	_storageLock.Lock()
	defer _storageLock.Unlock()
	encoded, err := json.Marshal(_storage)
	if err != nil {
		log.Errorf("error serializing app storage: %s", err)
		return
	}
	if err := os.WriteFile(_storageFile, encoded, 0644); err != nil {
		log.Errorf("error writing app storage: %s", err)
	}
}

func addKnownAccountToStorage(account Account) {
	_storageLock.Lock()
	accounts := []Account{account}
	seen := map[string]struct{}{account.Id: {}}
	for _, a := range _storage.KnownAccounts {
		if _, exists := seen[a.Id]; !exists {
			accounts = append(accounts, a)
			seen[a.Id] = struct{}{}
		}
	}
	_storage.KnownAccounts = accounts
	_storageLock.Unlock()
	go persistStorage()
}
