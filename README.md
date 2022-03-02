## WOWSPRESENCEGO
![WOWSPRESENCEGO_DISC](https://i.imgur.com/LPy7t6t.png)  

![WOWSPRESENCEGO_TERM](https://i.imgur.com/7Ly6m26.png)
### How does it work?
It reads the contents of `tempArenaInfo.json` file inside your WoWS replays directory and use that data to set your Discord's Rich Presence.  
The file `tempArenaInfo.json` is created/modified when you enter a battle and automatically deleted by the game after the battle.  
We leverage this behavior to detect the state of the client (in battle or not) instead of relying on mods to report the state.

### How to run it?

 1. Install [Go](https://go.dev/).
 2. Clone this repository.
 3. Build the executable via `go build .` or just run it `go run .`
