# countdown
Countdown is terminal based multi-event countdown timer. It uses the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framwework from [Charm_](https://charm.sh/).


## Installation
Just clone and build.
```
git clone https://github.com/aldernero/countdown.git
cd countdown
go build -o countdown main.go
```
When you launch it for the first time an `events.json` file will be created in the current directory, and you'll see a single event:
