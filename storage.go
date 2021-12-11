package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type AppStorage struct {
	sync.Mutex

	KnownAccounts []Account `json:"known_accounts"`

	LastUpdateCheckAt  time.Time `json:"last_update_check_at"`
	KnownLatestVersion string    `json:"known_latest_version"`
}

type Account struct {
	Id       string `json:"id"`
	Nickname string `json:"nickname"`
}

var (
	_storageFile string
	_storage     AppStorage
)

func storageInit() {
	_storageFile = filepath.Join(_internalDir, "storage.json")
	_storage.Load()
}

func (s *AppStorage) Load() {
	s.Lock()
	defer s.Unlock()
	encoded, err := os.ReadFile(_storageFile)
	if err != nil {
		log.Errorf("error loading storage.json: %s", err)
		return
	}
	if err := json.Unmarshal(encoded, &s); err != nil {
		log.Errorf("error parsing storage.json: %s", err)
		return
	}
}

func (s *AppStorage) Persist() {
	s.Lock()
	defer s.Unlock()
	encoded, err := json.Marshal(s)
	if err != nil {
		log.Errorf("error serializing app storage: %s", err)
		return
	}
	if err := os.WriteFile(_storageFile, encoded, 0644); err != nil {
		log.Errorf("error writing app storage: %s", err)
	}
}

func (s *AppStorage) AddKnownAccount(account Account) {
	s.Lock()
	accounts := []Account{account}
	seen := map[string]struct{}{account.Id: {}}
	for _, a := range s.KnownAccounts {
		if _, exists := seen[a.Id]; !exists {
			accounts = append(accounts, a)
			seen[a.Id] = struct{}{}
		}
	}
	s.KnownAccounts = accounts
	s.Unlock()
	go s.Persist()
}

func (s *AppStorage) SetUpdateCheck(latestVersion string) {
	s.Lock()
	s.LastUpdateCheckAt = time.Now()
	s.KnownLatestVersion = latestVersion
	s.Unlock()
	go s.Persist()
}
