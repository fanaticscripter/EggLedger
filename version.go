package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	version "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	_githubRepo          = "fanaticscripter/EggLedger"
	_updateCheckInterval = time.Hour * 23
)

func checkForUpdates() (newVersion string, err error) {
	wrap := func(err error) error {
		return errors.Wrap(err, "failed to check for new version")
	}
	runningVersion, err := version.NewVersion(_appVersion)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse running version %s", _appVersion)
		return "", wrap(err)
	}

	_storage.Lock()
	lastUpdateCheckAt := _storage.LastUpdateCheckAt
	knownLatestTag := _storage.KnownLatestVersion
	_storage.Unlock()
	if knownLatestTag != "" {
		if knownLatestVersion, err := version.NewVersion(knownLatestTag); err == nil {
			if knownLatestVersion.GreaterThan(runningVersion) {
				// A known new version is already stored, skip remote check.
				return knownLatestTag, nil
			}
		} else {
			log.Warnf("storage: failed to parse known_latest_version %s: %s", knownLatestTag, err)
		}
	}

	if time.Since(lastUpdateCheckAt) < _updateCheckInterval {
		log.Infof("%s since last update check, skipping", time.Since(lastUpdateCheckAt))
		return "", nil
	}

	latestTag, err := getLatestTag()
	if err != nil {
		return "", wrap(err)
	}
	log.Infof("latest tag: %s", latestTag)
	latestVersion, err := version.NewVersion(latestTag)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse latest version %s", latestTag)
		return "", wrap(err)
	}

	_storage.SetUpdateCheck(latestTag)

	if runningVersion.LessThan(latestVersion) {
		return latestTag, nil
	}
	return "", nil
}

func getLatestTag() (string, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	url := "https://api.github.com/repos/" + _githubRepo + "/releases/latest"
	resp, err := client.Get(url)
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", url)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "GET %s: %#v", url, string(body))
	}
	if resp.StatusCode != 200 {
		return "", errors.Errorf("GET %s: HTTP %d: %#v", url, resp.StatusCode, string(body))
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	err = json.Unmarshal(body, &release)
	if err != nil {
		return "", errors.Wrapf(err, "GET %s: %#v", url, string(body))
	}
	if release.TagName == "" {
		return "", errors.Errorf("GET %s: tag_name is empty: %#v", url, string(body))
	}
	return release.TagName, nil
}
