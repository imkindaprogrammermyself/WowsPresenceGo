package main

type Vehicle struct {
	ShipId   int    `json:"shipId"`
	Relation int    `json:"relation"`
	Id       int    `json:"id"`
	Name     string `json:"name"`
}

type TempArenaInfo struct {
	MatchGroup           string              `json:"matchGroup"`
	GameMode             int                 `json:"gameMode"`
	ClientVersionFromExe string              `json:"clientVersionFromExe"`
	ScenarioUiCategoryId int                 `json:"scenarioUiCategoryId"`
	MapDisplayName       string              `json:"mapDisplayName"`
	MapId                int                 `json:"mapId"`
	ClientVersionFromXml string              `json:"clientVersionFromXml"`
	WeatherParams        map[string][]string `json:"weatherParams"`
	DisabledShipClasses  interface{}         `json:"disabledShipClasses"`
	PlayersPerTeam       int                 `json:"playersPerTeam"`
	Duration             int                 `json:"duration"`
	GameLogic            string              `json:"gameLogic"`
	Name                 string              `json:"name"`
	Scenario             string              `json:"scenario"`
	PlayerID             int                 `json:"playerID"`
	Vehicles             []Vehicle           `json:"vehicles"`
	GameType             string              `json:"gameType"`
	DateTime             string              `json:"dateTime"`
	MapName              string              `json:"mapName"`
	PlayerName           string              `json:"playerName"`
	ScenarioConfigId     int                 `json:"scenarioConfigId"`
	TeamsCount           int                 `json:"teamsCount"`
	Logic                string              `json:"logic"`
	PlayerVehicle        string              `json:"playerVehicle"`
	BattleDuration       int                 `json:"battleDuration"`
	MapBorder            interface{}         `json:"mapBorder"`
}
