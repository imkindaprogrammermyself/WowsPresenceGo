package main

type GameInfo struct {
	Modes  map[string]string   `json:"modes"`
	Ships  map[string][]string `json:"ships"`
	Spaces map[string]string   `json:"spaces"`
}
