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

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/timer"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	defaultHelpHeight  = 5
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

// errStyle is package-level because countdownParser is called from
// Event.Description, which has no access to the model's styles.
var errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cError))

// getEventsFilePath returns the path to the events file in the user's config directory
func getEventsFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	appConfigDir := filepath.Join(configDir, appName)
	if err := os.MkdirAll(appConfigDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(appConfigDir, eventsFileName), nil
}

// styles holds every style used by the UI. Lip Gloss v2 removed
// AdaptiveColor, so styles are rebuilt whenever the terminal reports its
// background color.
type styles struct {
	app           lipgloss.Style
	title         lipgloss.Style
	detailTitle   lipgloss.Style
	inputTitle    lipgloss.Style
	selectedTitle lipgloss.Style
	selectedDesc  lipgloss.Style
	dimmedTitle   lipgloss.Style
	dimmedDesc    lipgloss.Style
	input         lipgloss.Style
	detail        lipgloss.Style
	focused       lipgloss.Style
	blurred       lipgloss.Style
	brightText    lipgloss.Style
	normalText    lipgloss.Style
	specialText   lipgloss.Style
	detailsLeft   lipgloss.Style
	detailsRight  lipgloss.Style
	help          lipgloss.Style
}

func newStyles(isDark bool) styles {
	lightDark := lipgloss.LightDark(isDark)
	itemTitle := lightDark(lipgloss.Color(cItemTitleLight), lipgloss.Color(cItemTitleDark))
	itemDesc := lightDark(lipgloss.Color(cItemDescLight), lipgloss.Color(cItemDescDark))
	brightText := lightDark(lipgloss.Color(cDimmedTitleLight), lipgloss.Color(cDimmedTitleDark))
	normalText := lightDark(lipgloss.Color(cDimmedDescLight), lipgloss.Color(cDimmedDescDark))
	dimmedDesc := lightDark(lipgloss.Color(cDimmedDescDark), lipgloss.Color(cDimmedDescLight))

	selectedTitle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(itemTitle).
		Foreground(itemTitle).
		Padding(0, 0, 0, 1)
	dimmedTitle := lipgloss.NewStyle().
		Foreground(brightText).
		Padding(0, 0, 0, 2)

	return styles{
		app: lipgloss.NewStyle().Margin(0, 1),
		title: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cTextLightGray)).
			Background(lipgloss.Color(cTitle)).
			Padding(0, 1),
		detailTitle: lipgloss.NewStyle().
			Width(defaultDetailWidth).
			Foreground(lipgloss.Color(cTextLightGray)).
			Background(lipgloss.Color(cDetailTitle)).
			Padding(0, 1).
			Align(lipgloss.Center),
		inputTitle: lipgloss.NewStyle().
			Width(defaultInputWidth).
			Foreground(lipgloss.Color(cTextLightGray)).
			Background(lipgloss.Color(cDetailTitle)).
			Padding(0, 1).
			Align(lipgloss.Center),
		selectedTitle: selectedTitle,
		selectedDesc:  selectedTitle.Foreground(itemDesc),
		dimmedTitle:   dimmedTitle,
		dimmedDesc:    dimmedTitle.Foreground(dimmedDesc),
		input: lipgloss.NewStyle().
			Margin(1, 1).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cPromptBorder)),
		detail: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.ThickBorder(), false, false, false, true).
			BorderForeground(itemTitle),
		focused: lipgloss.NewStyle().Foreground(lipgloss.Color(cPromptBorder)),
		blurred: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		brightText: lipgloss.NewStyle().
			Foreground(brightText),
		normalText: lipgloss.NewStyle().
			Foreground(normalText),
		specialText: lipgloss.NewStyle().
			Width(defaultDetailWidth).
			Margin(0, 0, 1, 0).
			Foreground(itemTitle).
			Align(lipgloss.Center),
		detailsLeft: lipgloss.NewStyle().
			Width(defaultDetailWidth / 2).
			Foreground(brightText).
			Align(lipgloss.Right),
		detailsRight: lipgloss.NewStyle().
			Width(defaultDetailWidth / 2).
			Foreground(normalText).
			Align(lipgloss.Left),
		help: list.DefaultStyles(isDark).HelpStyle.
			Width(defaultListWidth).
			Height(defaultHelpHeight),
	}
}

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
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q", "quit"),
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
	styles      styles
	isDark      bool
}

func NewMainModel() (MainModel, error) {
	m := MainModel{
		state: showEvents,
		timer: timer.New(timeout),
		// Assume a dark background until the terminal reports otherwise
		// via tea.BackgroundColorMsg.
		isDark: true,
	}
	m.styles = newStyles(m.isDark)
	events, err := readEventsFile()
	if err != nil {
		return m, err
	}
	items := make([]list.Item, len(events))
	for i := range events {
		items[i] = events[i]
	}
	m.inputs = make([]textinput.Model, 2)
	for i := range m.inputs {
		t := textinput.New()
		t.CharLimit = 30
		// In bubbles v2 an unset width truncates the placeholder, so size
		// the inputs to fit the longest placeholder.
		t.SetWidth(20)
		switch inputFields(i) {
		case inputNameField:
			t.Placeholder = "Event Name"
			t.Focus()
		case inputTimeField:
			t.Placeholder = "YYYY-MM-DD hh:mm:ss"
			t.CharLimit = 19
		}
		m.inputs[i] = t
	}
	m.events = list.New(items, m.newDelegate(), defaultListWidth, defaultListHeight)
	m.events.Title = "Events"
	m.events.SetShowPagination(true)
	m.applyStyles()
	if len(m.events.Items()) == 0 {
		m.state = noEvents
	}
	return m, nil
}

// newDelegate builds the list item delegate from the current styles.
func (m MainModel) newDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.Styles = list.NewDefaultItemStyles(m.isDark)
	delegate.Styles.SelectedTitle = m.styles.selectedTitle
	delegate.Styles.SelectedDesc = m.styles.selectedDesc
	delegate.Styles.DimmedTitle = m.styles.dimmedTitle
	delegate.Styles.DimmedDesc = m.styles.dimmedDesc
	delegate.ShortHelpFunc = func() []key.Binding { return []key.Binding{Keymap.Add, Keymap.Remove} }
	delegate.FullHelpFunc = func() [][]key.Binding { return [][]key.Binding{{Keymap.Add, Keymap.Remove}} }
	return delegate
}

// applyStyles pushes the current styles into the list and text inputs.
// Called at construction and again if the background color changes.
func (m *MainModel) applyStyles() {
	m.events.Styles.Title = m.styles.title
	m.events.Styles.HelpStyle = m.styles.help
	m.events.SetDelegate(m.newDelegate())
	for i := range m.inputs {
		s := textinput.DefaultStyles(m.isDark)
		s.Focused.Prompt = m.styles.focused
		s.Focused.Text = m.styles.focused
		m.inputs[i].SetStyles(s)
	}
}

func (m MainModel) Init() tea.Cmd {
	return tea.Batch(m.timer.Init(), tea.RequestBackgroundColor)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.isDark = msg.IsDark()
		m.styles = newStyles(m.isDark)
		m.applyStyles()
		return m, nil
	case tea.WindowSizeMsg:
		_, v := m.styles.app.GetFrameSize()
		m.events.SetSize(defaultListWidth, msg.Height-v)
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.state {
	case noEvents:
		if msg, ok := msg.(tea.KeyPressMsg); ok {
			switch {
			case key.Matches(msg, Keymap.Quit):
				return m, tea.Quit
			case key.Matches(msg, Keymap.Add):
				m.state = showInput
			}
		}
	case showEvents:
		if msg, ok := msg.(tea.KeyPressMsg); ok && m.events.FilterState() != list.Filtering {
			switch {
			case key.Matches(msg, Keymap.Quit):
				return m, tea.Quit
			case key.Matches(msg, Keymap.Add):
				m.state = showInput
			case key.Matches(msg, Keymap.Remove):
				m.events.RemoveItem(m.events.Index())
				if err := m.saveEventsToFile(); err != nil {
					cmds = append(cmds, m.events.NewStatusMessage(errStyle.Render(err.Error())))
				}
				if len(m.events.Items()) == 0 {
					m.state = noEvents
				}
			}
		}
		var cmd tea.Cmd
		m.events, cmd = m.events.Update(msg)
		cmds = append(cmds, cmd)
	case showInput:
		if msg, ok := msg.(tea.KeyPressMsg); ok {
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
					event, err := m.validateInputs()
					if err != nil {
						m.inputs[inputNameField].Reset()
						m.inputs[inputTimeField].Reset()
						m.focus = 0
						m.inputStatus = fmt.Sprintf("Error: %v", err)
						break
					}
					m.insertEvent(event)
					if err := m.saveEventsToFile(); err != nil {
						cmds = append(cmds, m.events.NewStatusMessage(errStyle.Render(err.Error())))
					}
					m.resetInputs()
					m.state = showEvents
				}
			}
		}
		cmds = append(cmds, m.updateFocus()...)
		for i := range m.inputs {
			var cmd tea.Cmd
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
	}
	var timerCmd tea.Cmd
	m.timer, timerCmd = m.timer.Update(msg)
	cmds = append(cmds, timerCmd)
	return m, tea.Batch(cmds...)
}

func (m MainModel) View() tea.View {
	var content string
	switch m.state {
	case noEvents:
		content = m.styles.input.Render("No events, add one with '+'")
	case showInput:
		content = m.inputView()
	default:
		listStr := m.styles.app.Render(m.events.View())
		content = lipgloss.JoinHorizontal(0.05, listStr, m.detailsString())
	}
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func main() {
	model, err := NewMainModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", appName, err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", appName, err)
		os.Exit(1)
	}
}

// insertEvent inserts an event into the list, keeping it sorted by time.
func (m *MainModel) insertEvent(event Event) {
	index := 0
	for _, item := range m.events.Items() {
		if event.Time >= item.(Event).Time {
			index++
		}
	}
	m.events.InsertItem(index, event)
}

func (m MainModel) detailsString() string {
	event, ok := m.events.SelectedItem().(Event)
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString(m.styles.detailTitle.Render(event.Name))
	b.WriteByte('\n')
	ts := time.Unix(event.Time, 0)
	b.WriteString(m.styles.normalText.Render("When (RFC1123): "))
	b.WriteString(m.styles.brightText.Render(ts.Format(time.RFC1123)))
	b.WriteByte('\n')
	b.WriteString(m.styles.normalText.Render("    When (ISO): "))
	b.WriteString(m.styles.brightText.Render(event.ToBasicString()))
	b.WriteString("\n\n\n")
	b.WriteString(m.styles.detailTitle.Render("Countdown"))
	b.WriteByte('\n')
	b.WriteString(m.styles.specialText.Render(countdownParser(event.Time)))
	b.WriteByte('\n')
	diff := time.Until(ts).Seconds()
	var left strings.Builder
	left.WriteString(strconv.FormatInt(int64(diff), 10))
	left.WriteByte('\n')
	left.WriteString(strconv.FormatFloat(diff/secondsPerMinute, 'f', 3, 64))
	left.WriteByte('\n')
	left.WriteString(strconv.FormatFloat(diff/secondsPerHour, 'f', 4, 64))
	left.WriteByte('\n')
	left.WriteString(strconv.FormatFloat(diff/secondsPerDay, 'f', 5, 64))
	left.WriteByte('\n')
	left.WriteString(strconv.FormatFloat(diff/secondsPerYear, 'f', 7, 64))
	right := " seconds\n minutes\n hours\n days\n years"
	return m.styles.detail.Render(b.String() +
		lipgloss.JoinHorizontal(lipgloss.Bottom, m.styles.detailsLeft.Render(left.String()), m.styles.detailsRight.Render(right)))
}

func countdownParser(ts int64) string {
	diff := int(time.Until(time.Unix(ts, 0)).Seconds())
	if diff < 0 {
		return errStyle.Render("Expired")
	}
	years := diff / secondsPerYear
	diff -= years * secondsPerYear
	days := diff / secondsPerDay
	diff -= days * secondsPerDay
	hours := diff / secondsPerHour
	diff -= hours * secondsPerHour
	minutes := diff / secondsPerMinute
	seconds := diff - minutes*secondsPerMinute
	switch {
	case years > 0:
		return fmt.Sprintf("%dy %dd %dh %dm %ds", years, days, hours, minutes, seconds)
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	case hours > 0:
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	case minutes > 0:
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}

func readEventsFile() ([]Event, error) {
	eventsFile, err := getEventsFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get events file path: %w", err)
	}

	data, err := os.ReadFile(eventsFile)
	if errors.Is(err, os.ErrNotExist) {
		events := []Event{nextGolangAnniversary()}
		return events, writeEventsFile(eventsFile, events)
	}
	if err != nil {
		return nil, err
	}
	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func writeEventsFile(path string, events []Event) error {
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (m MainModel) saveEventsToFile() error {
	eventsFile, err := getEventsFilePath()
	if err != nil {
		return fmt.Errorf("failed to get events file path: %w", err)
	}

	items := m.events.Items()
	events := make([]Event, len(items))
	for i, item := range items {
		events[i] = item.(Event)
	}
	return writeEventsFile(eventsFile, events)
}

func (m MainModel) inputView() string {
	var b strings.Builder
	b.WriteString(m.styles.inputTitle.Render("New Event"))
	b.WriteByte('\n')
	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	cancelStyle := m.styles.blurred
	if m.focus == int(inputCancelButton) {
		cancelStyle = m.styles.focused
	}
	submitStyle := m.styles.blurred
	if m.focus == int(inputSubmitButton) {
		submitStyle = m.styles.focused
	}
	fmt.Fprintf(
		&b,
		"\n\n%s  %s\n\n%s",
		cancelStyle.Render("[ Cancel ]"),
		submitStyle.Render("[ Submit ]"),
		errStyle.Render(m.inputStatus),
	)

	return m.styles.input.Render(b.String())
}

// updateFocus focuses the input matching m.focus and blurs the others.
// Focused/blurred styling is handled by the textinput styles themselves.
func (m *MainModel) updateFocus() []tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.inputs))
	for i := range m.inputs {
		if i == m.focus {
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}
	return cmds
}

func (m *MainModel) resetInputs() {
	m.inputs[inputNameField].Reset()
	m.inputs[inputTimeField].Reset()
	m.focus = 0
	m.inputStatus = ""
}

func (m MainModel) validateInputs() (Event, error) {
	var event Event
	name := m.inputs[inputNameField].Value()
	t := m.inputs[inputTimeField].Value()
	if name == "" || t == "" {
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
	if now.Before(thisYear) {
		return Event{nameStr, thisYear.Unix()}
	}
	return Event{nameStr, time.Date(year+1, 11, 10, 0, 0, 0, 0, time.Local).Unix()}
}
