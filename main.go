package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	secondsPerYear     = 31557600
	secondsPerDay      = 86400
	secondsPerHour     = 3600
	secondsPerMinute   = 60
	timeout            = 365 * 24 * time.Hour
	defaultListWidth   = 28
	defaultListHeight  = 40
	defaultDetailWidth = 45
	defaultInputWidth  = 22
	defaultHelpHeight  = 4
	appName            = "countdown"
	eventsFileName     = "events.json"
	inputTimeFormShort = "2006-01-02"
	inputTimeFormLong  = "2006-01-02 15:04:05"
	cError             = "#CF002E"
	cItemTitleDark     = "#F5EB6D"
	cItemTitleLight    = "#F3B512"
	cItemDescDark      = "#9E9742"
	cItemDescLight     = "#FFD975"
	cTitle             = "#2389D3"
	cDetailTitle       = "#D32389"
	cPromptBorder      = "#D32389"
	cDimmedTitleDark   = "#DDDDDD"
	cDimmedTitleLight  = "#222222"
	cDimmedDescDark    = "#999999"
	cDimmedDescLight   = "#555555"
	cTextLightGray     = "#FFFDF5"
)

// getEventsFilePath returns the path to the events file in the user's config directory
func getEventsFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	appConfigDir := filepath.Join(configDir, appName)
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(appConfigDir, eventsFileName), nil
}

var AppStyle = lipgloss.NewStyle().Margin(0, 1)
var TitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(cTextLightGray)).
	Background(lipgloss.Color(cTitle)).
	Padding(0, 1)
var DetailTitleStyle = lipgloss.NewStyle().
	Width(defaultDetailWidth).
	Foreground(lipgloss.Color(cTextLightGray)).
	Background(lipgloss.Color(cDetailTitle)).
	Padding(0, 1).
	Align(lipgloss.Center)
var InputTitleStyle = lipgloss.NewStyle().
	Width(defaultInputWidth).
	Foreground(lipgloss.Color(cTextLightGray)).
	Background(lipgloss.Color(cDetailTitle)).
	Padding(0, 1).
	Align(lipgloss.Center)
var SelectedTitle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderForeground(lipgloss.AdaptiveColor{Light: cItemTitleLight, Dark: cItemTitleDark}).
	Foreground(lipgloss.AdaptiveColor{Light: cItemTitleLight, Dark: cItemTitleDark}).
	Padding(0, 0, 0, 1)
var SelectedDesc = SelectedTitle.Copy().
	Foreground(lipgloss.AdaptiveColor{Light: cItemDescLight, Dark: cItemDescDark})
var DimmedTitle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: cDimmedTitleLight, Dark: cDimmedTitleDark}).
	Padding(0, 0, 0, 2)
var DimmedDesc = DimmedTitle.Copy().
	Foreground(lipgloss.AdaptiveColor{Light: cDimmedDescDark, Dark: cDimmedDescLight})
var InputStyle = lipgloss.NewStyle().
	Margin(1, 1).
	Padding(1, 2).
	Border(lipgloss.RoundedBorder(), true, true, true, true).
	BorderForeground(lipgloss.Color(cPromptBorder)).
	Render
var DetailStyle = lipgloss.NewStyle().
	Padding(1, 2).
	Border(lipgloss.ThickBorder(), false, false, false, true).
	BorderForeground(lipgloss.AdaptiveColor{Light: cItemTitleLight, Dark: cItemTitleDark}).
	Render
var ErrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cError)).Render
var NoStyle = lipgloss.NewStyle()
var FocusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cPromptBorder))
var BlurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
var BrightTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: cDimmedTitleLight, Dark: cDimmedTitleDark}).Render
var NormalTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: cDimmedDescLight, Dark: cDimmedDescDark}).Render
var SpecialTextStyle = lipgloss.NewStyle().
	Width(defaultDetailWidth).
	Margin(0, 0, 1, 0).
	Foreground(lipgloss.AdaptiveColor{Light: cItemTitleLight, Dark: cItemTitleDark}).
	Align(lipgloss.Center).Render
var DetailsBlockLeft = lipgloss.NewStyle().
	Width(defaultDetailWidth / 2).
	Foreground(lipgloss.AdaptiveColor{Light: cDimmedTitleLight, Dark: cDimmedTitleDark}).
	Align(lipgloss.Right).
	Render
var DetailsBlockRight = lipgloss.NewStyle().
	Width(defaultDetailWidth / 2).
	Foreground(lipgloss.AdaptiveColor{Light: cDimmedDescLight, Dark: cDimmedDescDark}).
	Align(lipgloss.Left).
	Render
var HelpStyle = list.DefaultStyles().HelpStyle.Width(defaultListWidth).Height(5)

type keymap struct {
	Add    key.Binding
	Remove key.Binding
	Next   key.Binding
	Prev   key.Binding
	Enter  key.Binding
	Back   key.Binding
	Quit   key.Binding
}

// Keymap reusable key mappings shared across models
var Keymap = keymap{
	Add: key.NewBinding(
		key.WithKeys("+"),
		key.WithHelp("+", "add"),
	),
	Remove: key.NewBinding(
		key.WithKeys("-"),
		key.WithHelp("-", "remove"),
	),
	Next: key.NewBinding(
		key.WithKeys("tab"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctlr+c", "q"),
		key.WithHelp("q", "back"),
	),
}

type sessionState int

const (
	showEvents sessionState = iota
	showInput
	noEvents
)

type inputFields int

const (
	inputNameField inputFields = iota
	inputTimeField
	inputCancelButton
	inputSubmitButton
)

type Event struct {
	Name string `json:"name"`
	Time int64  `json:"ts"`
}

func (e Event) ToBasicString() string {
	return time.Unix(e.Time, 0).String()
}

func (e Event) Title() string       { return e.Name }
func (e Event) Description() string { return countdownParser(e.Time) }
func (e Event) FilterValue() string { return e.Name }

type MainModel struct {
	state       sessionState
	focus       int
	events      list.Model
	inputs      []textinput.Model
	timer       timer.Model
	inputStatus string
}

func NewMainModel() MainModel {
	m := MainModel{
		state: showEvents,
		timer: timer.NewWithInterval(timeout, time.Second),
	}
	events, err := readEventsFile()
	if err != nil {
		panic(err)
	}
	items := make([]list.Item, len(events))
	for i := range events {
		items[i] = events[i]
	}
	m.inputs = make([]textinput.Model, 2)
	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.CharLimit = 30
		switch i {
		case 0:
			t.Placeholder = "Event Name"
			t.Focus()
			t.PromptStyle = FocusedStyle
			t.TextStyle = FocusedStyle
		case 1:
			t.Placeholder = "YYYY-MM-DD hh:mm:ss"
			t.CharLimit = 19
		}
		m.inputs[i] = t
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = SelectedTitle
	delegate.Styles.SelectedDesc = SelectedDesc
	delegate.Styles.DimmedTitle = DimmedTitle
	delegate.Styles.DimmedDesc = DimmedDesc
	delegate.ShortHelpFunc = func() []key.Binding { return []key.Binding{Keymap.Add, Keymap.Remove} }
	delegate.FullHelpFunc = func() [][]key.Binding { return [][]key.Binding{{Keymap.Add, Keymap.Remove}} }
	m.events = list.New(items, delegate, defaultListWidth, defaultListHeight)
	m.events.Title = "Events"
	m.events.Styles.Title = TitleStyle
	m.events.Styles.HelpStyle = HelpStyle
	m.events.SetShowPagination(true)
	if len(m.events.Items()) == 0 {
		m.state = noEvents
	}
	return m
}

func (m MainModel) Init() tea.Cmd {
	return m.timer.Init()
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch m.state {
	case noEvents:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, Keymap.Add):
				m.state = showInput
			}
		}
	case showEvents:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			_, v := AppStyle.GetFrameSize()
			m.events.SetSize(defaultListWidth, msg.Height-v)
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, Keymap.Quit):
				return m, tea.Quit
			case key.Matches(msg, Keymap.Add):
				m.state = showInput
			case key.Matches(msg, Keymap.Remove):
				m.events.RemoveItem(m.events.Index())
				if err := m.saveEventsToFile(); err != nil {
					panic(err)
				}
				if len(m.events.Items()) == 0 {
					m.state = noEvents
				}
			}
		}
		newEvents, newCmd := m.events.Update(msg)
		m.events = newEvents
		cmd = newCmd
	case showInput:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, Keymap.Back):
				m.resetInputs()
				m.state = showEvents
			case key.Matches(msg, Keymap.Next):
				m.focus++
				if m.focus > int(inputSubmitButton) {
					m.focus = int(inputNameField)
				}
			case key.Matches(msg, Keymap.Prev):
				m.focus--
				if m.focus < int(inputNameField) {
					m.focus = int(inputSubmitButton)
				}
			case key.Matches(msg, Keymap.Enter):
				switch inputFields(m.focus) {
				case inputNameField, inputTimeField:
					m.focus++
				case inputCancelButton:
					m.resetInputs()
					m.state = showEvents
				case inputSubmitButton:
					e, err := m.validateInputs()
					if err != nil {
						m.inputs[inputNameField].Reset()
						m.inputs[inputTimeField].Reset()
						m.focus = 0
						m.inputStatus = fmt.Sprintf("Error: %v", err)
						break
					}
					if len(m.events.Items()) == 0 {
						m.events.InsertItem(0, e)
					} else {
						index := 0
						for _, item := range m.events.Items() {
							if e.Time >= item.(Event).Time {
								index++
							}
						}
						m.events.InsertItem(index, e)
						if err := m.saveEventsToFile(); err != nil {
							panic(err)
						}
					}
					newEvents, newCmd := m.events.Update(msg)
					m.events = newEvents
					cmd = newCmd
					m.resetInputs()
					m.state = showEvents
				}
			}
		}
		cmds = append(cmds, m.updateInputs()...)
		for i := 0; i < len(m.inputs); i++ {
			newModel, cmd := m.inputs[i].Update(msg)
			m.inputs[i] = newModel
			cmds = append(cmds, cmd)
		}
	}
	timerModel, timerCmd := m.timer.Update(msg)
	m.timer = timerModel
	cmds = append(cmds, timerCmd)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	switch m.state {
	case noEvents:
		return InputStyle("No events, add one with '+'")
	case showInput:
		return m.inputView()
	default:
		listStr := AppStyle.Render(m.events.View())
		detailStr := m.detailsString()
		return lipgloss.JoinHorizontal(0.05, listStr, detailStr)
	}
}

func main() {
	p := tea.NewProgram(NewMainModel(), tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Printf("There was an error: %v", err)
		os.Exit(1)
	}
}

func (m MainModel) detailsString() string {
	var b strings.Builder
	event := m.events.SelectedItem().(Event)
	b.WriteString(DetailTitleStyle.Render(event.Name) + "\n")
	ts := time.Unix(event.Time, 0)
	rfc1123 := ts.Format(time.RFC1123)
	b.WriteString(NormalTextStyle("When (RFC1123): "))
	b.WriteString(BrightTextStyle(rfc1123) + "\n")
	b.WriteString(NormalTextStyle("    When (ISO): "))
	b.WriteString(BrightTextStyle(event.ToBasicString()) + "\n")
	b.WriteString("\n\n" + DetailTitleStyle.Render("Countdown") + "\n")
	b.WriteString(SpecialTextStyle(countdownParser(event.Time)) + "\n")
	diff := time.Until(ts).Seconds()
	seconds := int64(diff)
	minutes := diff / float64(secondsPerMinute)
	hours := diff / float64(secondsPerHour)
	days := diff / float64(secondsPerDay)
	years := diff / float64(secondsPerYear)
	var left strings.Builder
	left.WriteString(strconv.FormatInt(seconds, 10) + "\n")
	left.WriteString(strconv.FormatFloat(minutes, 'f', 3, 64) + "\n")
	left.WriteString(strconv.FormatFloat(hours, 'f', 4, 64) + "\n")
	left.WriteString(strconv.FormatFloat(days, 'f', 5, 64) + "\n")
	left.WriteString(strconv.FormatFloat(years, 'f', 7, 64))
	right := " seconds\n minutes\n hours\n days\n years"
	return DetailStyle(b.String() +
		lipgloss.JoinHorizontal(lipgloss.Bottom, DetailsBlockLeft(left.String()), DetailsBlockRight(right)))
}

func (m MainModel) eventsView() string {
	return AppStyle.Render(m.events.View())
}

func countdownParser(ts int64) string {
	t := time.Unix(ts, 0)
	diff := int(time.Until(t).Seconds())
	years := diff / secondsPerYear
	days := (diff - years*secondsPerYear) / secondsPerDay
	hours := (diff - years*secondsPerYear - days*secondsPerDay) / secondsPerHour
	minutes := (diff - years*secondsPerYear - days*secondsPerDay - hours*secondsPerHour) / secondsPerMinute
	seconds := diff - years*secondsPerYear - days*secondsPerDay - hours*secondsPerHour - minutes*secondsPerMinute
	var result string
	if years > 0 {
		result = fmt.Sprintf("%dy %dd %dh %dm %ds", years, days, hours, minutes, seconds)
	}
	if years == 0 && days > 0 {
		result = fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if years == 0 && days == 0 && hours > 0 {
		result = fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if years == 0 && days == 0 && hours == 0 && minutes > 0 {
		result = fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	if years == 0 && days == 0 && hours == 0 && minutes == 0 {
		result = fmt.Sprintf("%ds", seconds)
	}
	if diff < 0 {
		result = ErrStyle("Expired")
	}
	return result
}

func readEventsFile() ([]Event, error) {
	eventsFile, err := getEventsFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get events file path: %w", err)
	}

	var events []Event
	if _, err := os.Stat(eventsFile); errors.Is(err, os.ErrNotExist) {
		// create file
		_, err := os.Create(eventsFile)
		if err != nil {
			return events, err
		}
		event := nextGolangAnniversary()
		events = append(events, event)
		bytes, err := json.MarshalIndent(events, "", "  ")
		if err != nil {
			return events, err
		}
		err = os.WriteFile(eventsFile, bytes, 0644)
		return events, err
	}
	bytes, err := os.ReadFile(eventsFile)
	if err != nil {
		return events, err
	}
	err = json.Unmarshal(bytes, &events)
	if err != nil {
		return events, err
	}
	return events, nil
}

func (m MainModel) saveEventsToFile() error {
	eventsFile, err := getEventsFilePath()
	if err != nil {
		return fmt.Errorf("failed to get events file path: %w", err)
	}

	items := m.events.Items()
	events := make([]Event, len(items))
	for i := range items {
		events[i] = items[i].(Event)
	}
	bytes, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(eventsFile, bytes, 0644)
	return err
}

func (m MainModel) inputView() string {
	var b strings.Builder
	b.WriteString(InputTitleStyle.Render("New Event") + "\n")
	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	cancelButton := &BlurredStyle
	if m.focus == len(m.inputs) {
		cancelButton = &FocusedStyle
	}
	submitButton := &BlurredStyle
	if m.focus == len(m.inputs)+1 {
		submitButton = &FocusedStyle
	}
	_, err := fmt.Fprintf(
		&b,
		"\n\n%s  %s\n\n%s",
		cancelButton.Render("[ Cancel ]"),
		submitButton.Render("[ Submit ]"),
		ErrStyle(m.inputStatus),
	)
	if err != nil {
		fmt.Printf("Error formatting input string: %v\n", err)
		os.Exit(1)
	}

	return InputStyle(b.String())
}

func (m *MainModel) updateInputs() []tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := 0; i <= len(m.inputs)-1; i++ {
		if i == m.focus {
			// Set focused state
			cmds[i] = m.inputs[i].Focus()
			m.inputs[i].PromptStyle = FocusedStyle
			m.inputs[i].TextStyle = FocusedStyle
			continue
		}
		// Remove focused state
		m.inputs[i].Blur()
		m.inputs[i].PromptStyle = NoStyle
		m.inputs[i].TextStyle = NoStyle
	}
	return cmds
}

func (m MainModel) resetInputs() {
	m.inputs[inputNameField].Reset()
	m.inputs[inputTimeField].Reset()
	m.focus = 0
	m.inputStatus = ""
}

func (m MainModel) validateInputs() (Event, error) {
	var event Event
	name := m.inputs[0].Value()
	t := m.inputs[1].Value()
	if name == "" && t == "" {
		return event, fmt.Errorf("empty fields")
	}
	timeFormat := inputTimeFormLong
	if len(t) < len(inputTimeFormLong) {
		timeFormat = inputTimeFormShort
	}
	ts, err := time.ParseInLocation(timeFormat, t, time.Local)
	if err != nil {
		return event, err
	}
	if ts.Before(time.Now()) {
		return event, fmt.Errorf("event time is in the past")
	}
	event = Event{Name: name, Time: ts.Unix()}
	return event, nil
}

func nextGolangAnniversary() Event {
	nameStr := "Golang's Birthday"
	now := time.Now()
	year := now.Year()
	thisYear := time.Date(year, 11, 10, 0, 0, 0, 0, time.Local)
	nextYear := time.Date(year+1, 11, 10, 0, 0, 0, 0, time.Local)
	if now.Before(thisYear) {
		return Event{nameStr, thisYear.Unix()}
	}
	return Event{nameStr, nextYear.Unix()}
}
