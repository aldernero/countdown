# countdown
Countdown is terminal based multi-event countdown timer. It uses the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework from [Charm_](https://charm.sh/).



https://user-images.githubusercontent.com/96601789/182011443-15b35466-3969-490c-9f74-b30dcbd29a41.mp4



## Installation
Just clone and build.
```
git clone https://github.com/aldernero/countdown.git
cd countdown
go build -o countdown main.go
```
When you launch it for the first time an `events.json` file will be created in the current directory, and you'll see a single event:

![Screenshot_20220730_230038](https://user-images.githubusercontent.com/96601789/182010935-492b513e-4df4-48f8-8efb-28c1767ce2cb.png)

As you add and remove events, the `events.json` file will be updated.

## Usage

The controls are
- "+" to add an event
- "-" to remove an event
- "/" to filter events

The rest of the controls are what you would expect, up/down to traverse the list, tab to move between fields in the event input form.
