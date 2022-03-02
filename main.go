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

func fileWatcher(watcher *fsnotify.Watcher, chanTempArenaInfo chan *TempArenaInfo, chanIsBattleEnded chan bool) {
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				if filepath.Base(event.Name) != "tempArenaInfo.json" {
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
				if filepath.Base(event.Name) != "tempArenaInfo.json" {
					continue
				}
				chanIsBattleEnded <- true
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func processWatcher(procName string, delay time.Duration, isWowsRunning chan bool, watcher *fsnotify.Watcher) {
	replaysDir := ""
	isRemoved := false
	

	for {
		processes, err := process.Processes()
		isRunning := false

		if err != nil {
			continue
		}

		for _, process := range processes {
			processName, nameError := process.Name()
			cwd, _ := process.Cwd()

			if nameError != nil {
				continue
			}

			if processName == procName {
				if replaysDir == "" {
					replaysDir = fmt.Sprintf("%vreplays\\", cwd)
					isRemoved = false

					log.Printf("Process %v found...\n", processName)
					log.Printf("Setting replays directory to %v...\n", replaysDir)

					watcherAddErr := watcher.Add(replaysDir)
					if watcherAddErr != nil {
						log.Fatalf("Error at adding %v to watch list...\n", replaysDir)
					}
					isWowsRunning <- true
				}

				isRunning = true
				break
			}
		}

		if replaysDir != "" && !isRemoved && !isRunning{
			log.Printf("Process %v closed...\n", procName)
			isWowsRunning <- false
			watcherRemoveErr := watcher.Remove(replaysDir)

			if watcherRemoveErr == nil {
				replaysDir = ""
				isRemoved = true
			}
		}
		
		time.Sleep(delay)
	}
}

func main() {
	processName := "WorldOfWarships64.exe"
	scanRate    := time.Second*1

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

	go fileWatcher(watcher, chanTempArenaInfo, chanIsBattleEnded)
	go processEvents(chanIsWowsRunning, chanTempArenaInfo, chanIsBattleEnded)
	go processWatcher(processName, scanRate, chanIsWowsRunning, watcher)

	log.Printf("Waiting for %v...\n", processName)
	waitGroup.Wait()
}
