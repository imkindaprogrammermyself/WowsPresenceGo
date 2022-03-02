package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hugolgst/rich-go/client"
	"github.com/shirou/gopsutil/v3/process"
)

//go:embed gameInfo.json
var data []byte
var gameInfo = loadGameInfo()

func loadGameInfo() GameInfo {
	var gameInfo GameInfo

	unmarshallError := json.Unmarshal(data, &gameInfo)

	if unmarshallError != nil {
		log.Fatal("Corrupted gameInfo.json")
	}
	return gameInfo
}

func setRpcActivity(isLoggedIn bool, act client.Activity) {
	if !isLoggedIn {
		return
	}

	err := client.SetActivity(act)

	if err != nil {
		log.Println("Error at setting RPC activity.")
	}
}

func processEvents(chanIsWowsRunning chan bool, chanTempArenaInfo chan *TempArenaInfo, chanIsBattleEnded chan bool) {
	isLoggedIn := false

	for {
		select {
		case isWowsRunning := <-chanIsWowsRunning:
			if isWowsRunning {
				if !isLoggedIn {
					isLoggedIn = true
					err := client.Login("945234903392481330")

					if err != nil {
						log.Println("Discord RPC login error.")
						continue
					}
					log.Println("Setting your Discord Rich Presence...")
					now := time.Now()
					setRpcActivity(isLoggedIn, client.Activity{
						State:      "Idle",
						LargeImage: "idle",
						LargeText:  "Idle",
						Timestamps: &client.Timestamps{Start: &now},
					})
				}
			} else {
				if isLoggedIn {
					client.Logout()
					log.Println("Removing your Discord Rich Presence...")
					isLoggedIn = false
				}
			}

		case tempArenaInfo := <-chanTempArenaInfo:
			now := time.Now()
			end := now.Add(time.Second * time.Duration(tempArenaInfo.Duration+30))
			ship := gameInfo.Ships[strings.Split(tempArenaInfo.PlayerVehicle, "-")[0]]
			species, tier, name := ship[0], ship[1], ship[2]
			species_lower := strings.ToLower(species)

			mapName := gameInfo.Spaces["IDS_"+strings.ToUpper(tempArenaInfo.MapName)]
			vehicle := fmt.Sprintf("%v %v", tier, name)
			log.Printf("Battle has started. (%v, %v, %v)", mapName, species, vehicle)

			setRpcActivity(isLoggedIn, client.Activity{
				LargeImage: strings.ToLower(tempArenaInfo.GameType),
				LargeText:  gameInfo.Modes[fmt.Sprintf("IDS_GAMEMODE_%v_TITLE", strings.ToUpper(tempArenaInfo.GameLogic))],
				SmallImage: species_lower,
				SmallText:  species,
				Details:    fmt.Sprintf("on %v", mapName),
				State:      fmt.Sprintf("Playing %v", vehicle),
				Timestamps: &client.Timestamps{Start: &now, End: &end},
			})
		case isBattleEnded := <-chanIsBattleEnded:
			if isBattleEnded {
				log.Println("Battle has ended...")
			}

			now := time.Now()
			setRpcActivity(isLoggedIn, client.Activity{
				State:      "Idle",
				LargeImage: "idle",
				LargeText:  "Idle",
				Timestamps: &client.Timestamps{Start: &now},
			})
		}

	}
}

func fileWatcher(filename string, watcher *fsnotify.Watcher, chanTempArenaInfo chan *TempArenaInfo, chanIsBattleEnded chan bool) {
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				if filepath.Base(event.Name) != filename {
					continue
				}

				file, err := os.Open(event.Name)
				if err != nil {
					log.Println(err)
					file.Close()
					continue
				}

				data, readError := ioutil.ReadAll(file)

				if readError != nil {
					continue
				}

				var tempArenaInfo TempArenaInfo
				unmarshalError := json.Unmarshal(data, &tempArenaInfo)

				if unmarshalError != nil {
					continue
				}

				file.Close()

				chanTempArenaInfo <- &tempArenaInfo
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				if filepath.Base(event.Name) != filename {
					continue
				}
				chanIsBattleEnded <- true
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func isRunning(processName string) *process.Process {
	processes, err := process.Processes()
	if err != nil {
		return nil
	}
	for _, process := range processes {
		name, nameErr := process.Name()
		if nameErr != nil {
			return nil
		}
		if name == processName {
			return process
		}
	}
	return nil
}

func processWatcher(processName string, checkFrequency time.Duration) chan *process.Process {
	chanProcess := make(chan *process.Process)
	go func() {
		isSent := false
		for {
			process := isRunning(processName)
			if process != nil && !isSent {
				isSent = true
				chanProcess <- process
			} else if process == nil && isSent {
				isSent = false
				chanProcess <- nil
			}
			time.Sleep(checkFrequency)
		}
	}()
	return chanProcess
}

func main() {
	processName := "WorldOfWarships64.exe"
	fileToWatch := "tempArenaInfo.json"
	scanRate := time.Second * 1
	
	fmt.Println("*********************************")
	fmt.Println("*** Welcome to WowsPresenceGo ***")
	fmt.Println("*********************************")
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)

	chanTempArenaInfo, chanIsBattleEnded, chanIsWowsRunning := make(chan *TempArenaInfo), make(chan bool), make(chan bool)
	watcher, watcherError := fsnotify.NewWatcher()

	if watcherError != nil {
		log.Fatal(watcherError)
	}
	defer watcher.Close()
	chanProcess := processWatcher(processName, scanRate)

	go func() {
		replaysDir := ""
		for p := range chanProcess {
			if p != nil {
				cwd, cwdError := p.Cwd()
				if cwdError != nil {
					continue
				}
				replaysDir := fmt.Sprintf("%vreplays\\", cwd)
				watcherAddErr := watcher.Add(replaysDir)
				if watcherAddErr != nil {
					log.Fatalf("Error at adding %v to watch list...\n", replaysDir)
				}
				log.Printf("Process %v found...\n", processName)
				log.Printf("Setting replays directory to %v...\n", replaysDir)
				chanIsWowsRunning <- true
			} else {
				if replaysDir != "" {
					replaysDir = ""
					watcher.Remove(replaysDir)
				}
				log.Printf("Process %v closed...\n", processName)
				chanIsWowsRunning <- false
			}
		}
	}()

	go fileWatcher(fileToWatch, watcher, chanTempArenaInfo, chanIsBattleEnded)
	go processEvents(chanIsWowsRunning, chanTempArenaInfo, chanIsBattleEnded)

	log.Printf("Waiting for %v...\n", processName)
	waitGroup.Wait()
}
