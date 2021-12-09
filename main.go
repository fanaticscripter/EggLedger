package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fanaticscripter/EggLedger/db"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
	"github.com/skratchdot/open-golang/open"
	"github.com/zserge/lorca"
	"golang.org/x/sync/semaphore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	//go:embed VERSION
	_appVersion string

	//go:embed www
	_fs embed.FS

	_rootDir     string
	_internalDir string

	_appIsInForbiddenDirectory bool

	// macOS Gateway security feature which executed apps with xattr
	// com.apple.quarantine in certain locations (like ~/Downloads) in a jailed
	// readonly environment. The jail looks like:
	// /private/var/folders/<...>/<...>/T/AppTranslocation/<UUID>/d/internal
	_appIsTranslocated bool

	_devMode = os.Getenv("DEV_MODE") != ""
)

const (
	_requestInterval = 3 * time.Second
)

type UI struct {
	lorca.UI
}

func (u UI) MustLoad(url string) {
	err := u.Load(url)
	if err != nil {
		log.Fatal(err)
	}
}

func (u UI) MustBind(name string, f interface{}) {
	err := u.Bind(name, f)
	if err != nil {
		log.Fatal(err)
	}
}

type AppState string

//nolint:deadcode
const (
	AppState_AWAITING_INPUT    AppState = "AwaitingInput"
	AppState_FETCHING_SAVE     AppState = "FetchingSave"
	AppState_FETCHING_MISSIONS AppState = "FetchingMissions"
	AppState_EXPORTING_DATA    AppState = "ExportingData"
	AppState_SUCCESS           AppState = "Success"
	AppState_FAILED            AppState = "Failed"
	AppState_INTERRUPTED       AppState = "Interrupted"
)

type MissionProgress struct {
	Total                   int     `json:"total"`
	Finished                int     `json:"finished"`
	FinishedPercentage      string  `json:"finishedPercentage"`
	ExpectedFinishTimestamp float64 `json:"expectedFinishTimestamp"`
}

type worker struct {
	*semaphore.Weighted
	ctx     context.Context
	cancel  context.CancelFunc
	ctxlock sync.Mutex
}

func init() {
	log.SetLevel(log.InfoLevel)
	// Send a copy of logs to $TMPDIR/EggLedger.log in case the app crashes
	// before we can even set up persistent logging.
	tmplog, err := os.OpenFile(filepath.Join(os.TempDir(), "EggLedger.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Error(err)
	}
	log.AddHook(&writer.Hook{
		Writer:    tmplog,
		LogLevels: log.AllLevels,
	})

	path, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		log.Fatal(err)
	}
	_rootDir = filepath.Dir(path)
	if runtime.GOOS == "darwin" {
		// Locate parent dir of app bundle if we're inside a Mac app.
		parent, dir1 := filepath.Split(_rootDir)
		parent = filepath.Clean(parent)
		parent, dir2 := filepath.Split(parent)
		parent = filepath.Clean(parent)
		if dir1 == "MacOS" && dir2 == "Contents" && strings.HasSuffix(parent, ".app") {
			_rootDir = filepath.Dir(parent)
		}
	}
	log.Infof("root dir: %s", _rootDir)

	_internalDir = filepath.Join(_rootDir, "internal")

	// Make sure the app isn't located in the system/user app directory or
	// downloads dir.
	if runtime.GOOS == "darwin" {
		if _rootDir == "/Applications" {
			_appIsInForbiddenDirectory = true
		} else {
			pattern := regexp.MustCompile(`^/Users/[^/]+/(Applications|Downloads)$`)
			if pattern.MatchString(_rootDir) {
				_appIsInForbiddenDirectory = true
			}
		}
	} else {
		// On non-macOS platforms, just check whether the root dir ends in "/Downloads".
		pattern := regexp.MustCompile(`[\/]Downloads$`)
		if pattern.MatchString(_rootDir) {
			_appIsInForbiddenDirectory = true
		}
	}
	if _appIsInForbiddenDirectory {
		log.Error("app is in a forbidden directory")
		return
	}

	if runtime.GOOS == "darwin" {
		if strings.HasPrefix(_rootDir, "/private/var/folders/") {
			_appIsTranslocated = true
		}
	}
	if _appIsTranslocated {
		log.Error("app is translocated")
		return
	}

	if err := os.MkdirAll(_internalDir, 0755); err != nil {
		log.Fatal(err)
	}
	if err := hide(_internalDir); err != nil {
		log.Errorf("error hiding internal directory: %s", err)
	}

	// Set up persistent logging.
	logdir := filepath.Join(_rootDir, "logs")
	if err := os.MkdirAll(logdir, 0755); err != nil {
		log.Error(err)
	} else {
		logfile := filepath.Join(logdir, "app.log")
		logger := &lumberjack.Logger{
			Filename:  logfile,
			MaxSize:   5, // megabytes
			MaxAge:    7, // days
			LocalTime: true,
			Compress:  true,
		}
		log.AddHook(&writer.Hook{
			Writer:    logger,
			LogLevels: log.AllLevels,
		})
	}

	storageInit()
	dataInit()
}

func main() {
	if _devMode {
		log.Info("starting app in dev mode")
	}

	chrome := lorca.LocateChrome()
	if chrome == "" {
		lorca.PromptDownload()
		log.Fatal("unable to locate Chrome")
		return
	}

	args := []string{}
	if runtime.GOOS == "linux" {
		args = append(args, "--class=Lorca")
	}
	u, err := lorca.New("", "", 600, 600, args...)
	if err != nil {
		log.Fatal(err)
	}
	ui := UI{u}
	defer ui.Close()

	updateKnownAccounts := func(accounts []Account) {
		encoded, err := json.Marshal(accounts)
		if err != nil {
			log.Error(err)
			return
		}
		ui.Eval(fmt.Sprintf("window.updateKnownAccounts(%s)", encoded))
	}
	updateState := func(state AppState) {
		ui.Eval(fmt.Sprintf("window.updateState('%s')", state))
	}
	updateMissionProgress := func(progress MissionProgress) {
		encoded, err := json.Marshal(progress)
		if err != nil {
			log.Error(err)
			return
		}
		ui.Eval(fmt.Sprintf("window.updateMissionProgress(%s)", encoded))
	}
	updateExportedFiles := func(files []string) {
		encoded, err := json.Marshal(files)
		if err != nil {
			log.Error(err)
			return
		}
		ui.Eval(fmt.Sprintf("window.updateExportedFiles(%s)", encoded))
	}
	emitMessage := func(message string, isError bool) {
		encoded, err := json.Marshal(message)
		if err != nil {
			log.Error(err)
			return
		}
		ui.Eval(fmt.Sprintf("window.emitMessage(%s, %t)", encoded, isError))
	}

	pinfo := func(args ...interface{}) {
		log.Info(args...)
		emitMessage(fmt.Sprint(args...), false)
	}
	perror := func(args ...interface{}) {
		log.Error(args...)
		emitMessage(fmt.Sprint(args...), true)
	}

	ui.MustBind("appVersion", func() string {
		return _appVersion
	})

	ui.MustBind("appDirectory", func() string {
		return _rootDir
	})

	ui.MustBind("appIsInForbiddenDirectory", func() bool {
		return _appIsInForbiddenDirectory
	})

	ui.MustBind("appIsTranslocated", func() bool {
		return _appIsTranslocated
	})

	ui.MustBind("knownAccounts", func() []Account {
		_storageLock.Lock()
		defer _storageLock.Unlock()
		return _storage.KnownAccounts
	})

	w := &worker{
		Weighted: semaphore.NewWeighted(1),
	}
	ui.MustBind("fetchPlayerData", func(playerId string) {
		go func() {
			if !w.TryAcquire(1) {
				perror("already fetching player data, cannot accept new work")
				return
			}
			defer w.Release(1)

			ctx, cancel := context.WithCancel(context.Background())
			w.ctxlock.Lock()
			w.ctx = ctx
			w.cancel = cancel
			w.ctxlock.Unlock()

			checkInterrupt := func() bool {
				select {
				case <-ctx.Done():
					perror("interrupted")
					updateState(AppState_INTERRUPTED)
					return true
				default:
					return false
				}
			}

			updateState(AppState_FETCHING_SAVE)
			fc, err := fetchFirstContactWithContext(w.ctx, playerId)
			if err != nil {
				perror(err)
				if !checkInterrupt() {
					updateState(AppState_FAILED)
				}
				return
			}
			nickname := fc.GetBackup().GetUserName()
			msg := fmt.Sprintf("successfully fetched backup for %s", playerId)
			if nickname != "" {
				msg += fmt.Sprintf(" (%s)", nickname)
			}
			pinfo(msg)
			lastBackupTime := fc.GetBackup().GetSettings().GetLastBackupTime()
			if lastBackupTime != 0 {
				t := unixToTime(lastBackupTime)
				now := time.Now()
				if t.After(now) {
					t = now
				}
				msg := fmt.Sprintf("backup is from %s", humanize.Time(t))
				pinfo(msg)
			} else {
				perror("backup is from unknown time")
			}
			addKnownAccountToStorage(Account{Id: playerId, Nickname: nickname})
			_storageLock.Lock()
			updateKnownAccounts(_storage.KnownAccounts)
			_storageLock.Unlock()
			if checkInterrupt() {
				return
			}

			missions := fc.GetCompletedMissions()
			existingMissionIds, err := db.RetrievePlayerCompleteMissionIds(playerId)
			if err != nil {
				perror(err)
				updateState(AppState_FAILED)
				return
			}
			seen := make(map[string]struct{})
			for _, id := range existingMissionIds {
				seen[id] = struct{}{}
			}
			var newMissionIds []string
			var newMissionStartTimestamps []float64
			for _, mission := range missions {
				id := mission.GetIdentifier()
				if _, exists := seen[id]; !exists {
					newMissionIds = append(newMissionIds, id)
					newMissionStartTimestamps = append(newMissionStartTimestamps, mission.GetStartTimeDerived())
				}
			}
			pinfo(fmt.Sprintf("found %d completed missions, need to fetch %d",
				len(missions), len(newMissionIds)))

			total := len(newMissionIds)
			if total > 0 {
				updateState(AppState_FETCHING_MISSIONS)
				reportProgress := func(finished int) {
					updateMissionProgress(MissionProgress{
						Total:                   total,
						Finished:                finished,
						FinishedPercentage:      fmt.Sprintf("%.1f%%", float64(finished)/float64(total)*100),
						ExpectedFinishTimestamp: timeToUnix(time.Now().Add(time.Duration(total-finished) * _requestInterval)),
					})
				}
				reportProgress(0)
				finishedCh := make(chan struct{}, total)
				go func() {
					finished := 0
					for range finishedCh {
						finished++
						reportProgress(finished)
					}
				}()
				errored := 0
				var wg sync.WaitGroup
			MissionsLoop:
				for i := 0; i < total; i++ {
					if i != 0 {
						select {
						case <-ctx.Done():
							break MissionsLoop
						case <-time.After(_requestInterval):
						}
					}
					wg.Add(1)
					go func(missionId string, startTimestamp float64) {
						defer wg.Done()
						_, err := fetchCompleteMissionWithContext(w.ctx, playerId, missionId, startTimestamp)
						if err != nil {
							perror(err)
							errored++
						}
						finishedCh <- struct{}{}
					}(newMissionIds[i], newMissionStartTimestamps[i])
				}
				wg.Wait()
				close(finishedCh)
				if checkInterrupt() {
					return
				}
				if errored > 0 {
					perror(fmt.Sprintf("%d of %d missions failed to fetch", errored, total))
					updateState(AppState_FAILED)
					return
				} else {
					pinfo(fmt.Sprintf("successfully fetched %d missions", total))
				}
			}

			updateState(AppState_EXPORTING_DATA)
			completeMissions, err := db.RetrievePlayerCompleteMissions(playerId)
			if err != nil {
				perror(err)
				updateState(AppState_FAILED)
				return
			}
			var exportMissions []*mission
			for _, m := range completeMissions {
				exportMissions = append(exportMissions, newMission(m))
			}
			if checkInterrupt() {
				return
			}

			exportDir := filepath.Join(_rootDir, "exports", "missions")
			if err := os.MkdirAll(exportDir, 0755); err != nil {
				perror(errors.Wrap(err, "failed to create export directory"))
				updateState(AppState_FAILED)
				return
			}
			filenameTimestamp := time.Now().Format("20060102_150405")

			xlsxFile := filepath.Join(exportDir, playerId+"."+filenameTimestamp+".xlsx")
			xlsxFileRel, _ := filepath.Rel(_rootDir, xlsxFile)
			if err := exportMissionsToXlsx(exportMissions, xlsxFile); err != nil {
				perror(err)
				updateState(AppState_FAILED)
				return
			}
			if checkInterrupt() {
				return
			}

			csvFile := filepath.Join(exportDir, playerId+"."+filenameTimestamp+".csv")
			csvFileRel, _ := filepath.Rel(_rootDir, csvFile)
			if err := exportMissionsToCsv(exportMissions, csvFile); err != nil {
				perror(err)
				updateState(AppState_FAILED)
				return
			}
			if checkInterrupt() {
				return
			}

			updateExportedFiles([]string{xlsxFileRel, csvFileRel})

			pinfo("done.")
			updateState(AppState_SUCCESS)
		}()
	})

	ui.MustBind("stopFetchingPlayerData", func() {
		w.ctxlock.Lock()
		defer w.ctxlock.Unlock()
		if w.cancel != nil {
			w.cancel()
		}
	})

	ui.MustBind("openFile", func(file string) {
		path := filepath.Join(_rootDir, file)
		if err := open.Start(path); err != nil {
			log.Errorf("opening %s: %s", path, err)
		}
	})

	ui.MustBind("openFileInFolder", func(file string) {
		path := filepath.Join(_rootDir, file)
		if err := openFolderAndSelect(path); err != nil {
			log.Errorf("opening %s in folder: %s", path, err)
		}
	})

	ui.MustBind("openURL", func(url string) {
		if err := open.Start(url); err != nil {
			log.Errorf("opening %s: %s", url, err)
		}
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	go func() {
		var httpfs http.FileSystem
		if _devMode {
			httpfs = http.Dir("www")
		} else {
			wwwfs, err := fs.Sub(_fs, "www")
			if err != nil {
				log.Fatal(err)
			}
			httpfs = http.FS(wwwfs)
		}
		err := http.Serve(ln, http.FileServer(httpfs))
		if err != nil {
			log.Fatal(err)
		}
	}()
	ui.MustLoad(fmt.Sprintf("http://%s/", ln.Addr()))

	// Wait until the interrupt signal arrives or browser window is closed.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	select {
	case <-sigc:
	case <-ui.Done():
	}
}
