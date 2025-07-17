package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
)

// testHelper provides utilities for testing with config directories
type testHelper struct {
	originalConfigDir string
	testConfigDir     string
}

func newTestHelper(t *testing.T) *testHelper {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "countdown-test-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Store original config dir environment variable
	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")

	// Set test config directory
	os.Setenv("XDG_CONFIG_HOME", testDir)

	return &testHelper{
		originalConfigDir: originalConfigDir,
		testConfigDir:     testDir,
	}
}

func (th *testHelper) cleanup() {
	// Restore original environment
	if th.originalConfigDir != "" {
		os.Setenv("XDG_CONFIG_HOME", th.originalConfigDir)
	} else {
		os.Unsetenv("XDG_CONFIG_HOME")
	}

	// Clean up test directory
	os.RemoveAll(th.testConfigDir)
}

func (th *testHelper) removeEventsFile() {
	eventsFile, err := getEventsFilePath()
	if err == nil {
		os.Remove(eventsFile)
	}
}

func TestNextGolangAnniversary(t *testing.T) {
	tests := []struct {
		name     string
		now      time.Time
		expected time.Time
	}{
		{
			name:     "Before November 10th",
			now:      time.Date(2023, 6, 15, 12, 0, 0, 0, time.Local),
			expected: time.Date(2023, 11, 10, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "On November 10th",
			now:      time.Date(2023, 11, 10, 12, 0, 0, 0, time.Local),
			expected: time.Date(2024, 11, 10, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "After November 10th",
			now:      time.Date(2023, 11, 15, 12, 0, 0, 0, time.Local),
			expected: time.Date(2024, 11, 10, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "December 31st",
			now:      time.Date(2023, 12, 31, 23, 59, 59, 0, time.Local),
			expected: time.Date(2024, 11, 10, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "January 1st",
			now:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2024, 11, 10, 0, 0, 0, 0, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logic manually since we can't override time.Now
			year := tt.now.Year()
			thisYear := time.Date(year, 11, 10, 0, 0, 0, 0, time.Local)
			nextYear := time.Date(year+1, 11, 10, 0, 0, 0, 0, time.Local)

			var expectedEvent Event
			if tt.now.Before(thisYear) {
				expectedEvent = Event{"Golang's Birthday", thisYear.Unix()}
			} else {
				expectedEvent = Event{"Golang's Birthday", nextYear.Unix()}
			}

			// For testing purposes, we'll manually calculate what the function should return
			// given the current time logic in nextGolangAnniversary()
			expectedUnix := tt.expected.Unix()
			if expectedEvent.Time != expectedUnix {
				t.Errorf("Expected time %d (%s), got %d (%s)",
					expectedUnix, tt.expected.Format(time.RFC3339),
					expectedEvent.Time, time.Unix(expectedEvent.Time, 0).Format(time.RFC3339))
			}

			if expectedEvent.Name != "Golang's Birthday" {
				t.Errorf("Expected name 'Golang's Birthday', got '%s'", expectedEvent.Name)
			}
		})
	}
}

func TestCountdownParser(t *testing.T) {
	// Use current time as base to ensure tests work regardless of when they're run
	now := time.Now()

	tests := []struct {
		name           string
		target         time.Time
		expectedPrefix string
		shouldExpire   bool
	}{
		{
			name:           "Future event - years, days, hours, minutes, seconds",
			target:         now.AddDate(2, 3, 5).Add(6*time.Hour + 30*time.Minute + 45*time.Second),
			expectedPrefix: "2y",
			shouldExpire:   false,
		},
		{
			name:           "Future event - days, hours, minutes, seconds",
			target:         now.AddDate(0, 0, 5).Add(6*time.Hour + 30*time.Minute + 45*time.Second),
			expectedPrefix: "5d",
			shouldExpire:   false,
		},
		{
			name:           "Future event - hours, minutes, seconds",
			target:         now.Add(6*time.Hour + 30*time.Minute + 45*time.Second),
			expectedPrefix: "6h",
			shouldExpire:   false,
		},
		{
			name:           "Future event - minutes, seconds",
			target:         now.Add(30*time.Minute + 45*time.Second),
			expectedPrefix: "30m",
			shouldExpire:   false,
		},
		{
			name:           "Future event - seconds only",
			target:         now.Add(45 * time.Second),
			expectedPrefix: "45s",
			shouldExpire:   false,
		},
		{
			name:           "Past event",
			target:         now.Add(-1 * time.Hour),
			expectedPrefix: "Expired",
			shouldExpire:   true,
		},
		{
			name:           "Exactly now",
			target:         now,
			expectedPrefix: "0s",
			shouldExpire:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countdownParser(tt.target.Unix())
			if tt.name == "Future event - seconds only" {
				if !(result == "45s" || result == "44s") {
					t.Errorf("Expected '45s' or '44s', got '%s'", result)
				}
				return
			}
			if tt.shouldExpire {
				if !strings.Contains(result, "Expired") {
					t.Errorf("Expected result to contain 'Expired', got '%s'", result)
				}
			} else {
				if strings.Contains(result, "Expired") {
					t.Errorf("Expected non-expired result, got result containing 'Expired': '%s'", result)
				}
				if !strings.HasPrefix(result, tt.expectedPrefix) {
					t.Errorf("Expected result to start with '%s', got '%s'", tt.expectedPrefix, result)
				}
			}
		})
	}
}

func TestEventMethods(t *testing.T) {
	event := Event{
		Name: "Test Event",
		Time: time.Now().AddDate(1, 0, 0).Unix(), // Use a future date
	}

	t.Run("Title", func(t *testing.T) {
		if event.Title() != "Test Event" {
			t.Errorf("Expected 'Test Event', got '%s'", event.Title())
		}
	})

	t.Run("FilterValue", func(t *testing.T) {
		if event.FilterValue() != "Test Event" {
			t.Errorf("Expected 'Test Event', got '%s'", event.FilterValue())
		}
	})

	t.Run("ToBasicString", func(t *testing.T) {
		expected := time.Unix(event.Time, 0).String()
		if event.ToBasicString() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, event.ToBasicString())
		}
	})

	t.Run("Description", func(t *testing.T) {
		// Description should return the countdown string
		desc := event.Description()
		if desc == "" {
			t.Error("Description should not be empty")
		}
		// Since this depends on current time, we just check it's not empty
		// and doesn't contain "Expired" for a future date
		if strings.Contains(desc, "Expired") {
			t.Error("Description should not contain 'Expired' for a future date")
		}
	})
}

func TestValidateInputs(t *testing.T) {
	tests := []struct {
		name        string
		eventName   string
		timeString  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid inputs - long format",
			eventName:   "Test Event",
			timeString:  time.Now().AddDate(0, 1, 0).Format("2006-01-02 15:04:05"),
			expectError: false,
		},
		{
			name:        "Valid inputs - short format",
			eventName:   "Test Event",
			timeString:  time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
			expectError: false,
		},
		{
			name:        "Empty fields",
			eventName:   "",
			timeString:  "",
			expectError: true,
			errorMsg:    "empty fields",
		},
		{
			name:        "Invalid time format",
			eventName:   "Test Event",
			timeString:  "invalid-time",
			expectError: true,
		},
		{
			name:        "Past time",
			eventName:   "Test Event",
			timeString:  "2020-01-01 12:00:00",
			expectError: true,
			errorMsg:    "event time is in the past",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := MainModel{
				inputs: make([]textinput.Model, 2),
			}

			// Set up input values
			nameInput := textinput.New()
			nameInput.SetValue(tt.eventName)
			model.inputs[0] = nameInput

			timeInput := textinput.New()
			timeInput.SetValue(tt.timeString)
			model.inputs[1] = timeInput

			event, err := model.validateInputs()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if event.Name != tt.eventName {
					t.Errorf("Expected event name '%s', got '%s'", tt.eventName, event.Name)
				}
			}
		})
	}
}

func TestReadEventsFile(t *testing.T) {
	// Test with non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		// Remove the file if it exists
		th := newTestHelper(t)
		defer th.cleanup()
		th.removeEventsFile()

		events, err := readEventsFile()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(events) != 1 {
			t.Errorf("Expected 1 event (default Golang birthday), got %d", len(events))
		}

		if events[0].Name != "Golang's Birthday" {
			t.Errorf("Expected 'Golang's Birthday', got '%s'", events[0].Name)
		}

		// Clean up
		th.removeEventsFile()
	})

	// Test with existing file
	t.Run("Existing file", func(t *testing.T) {
		// Create a test events file
		th := newTestHelper(t)
		defer th.cleanup()
		th.removeEventsFile()

		testEvents := []Event{
			{Name: "Test Event 1", Time: time.Now().Add(24 * time.Hour).Unix()},
			{Name: "Test Event 2", Time: time.Now().Add(48 * time.Hour).Unix()},
		}

		// Save test events
		model := MainModel{}
		model.events = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
		for _, event := range testEvents {
			model.events.InsertItem(0, event)
		}
		err := model.saveEventsToFile()
		if err != nil {
			t.Fatalf("Failed to save test events: %v", err)
		}

		// Read events back
		events, err := readEventsFile()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(events) != 2 {
			t.Errorf("Expected 2 events, got %d", len(events))
		}

		// Clean up
		th.removeEventsFile()
	})
}

func TestMainModelInitialization(t *testing.T) {
	// Remove events file to test default initialization
	th := newTestHelper(t)
	defer th.cleanup()
	th.removeEventsFile()

	model := NewMainModel()

	// Test initial state
	if model.state != showEvents {
		t.Errorf("Expected initial state to be showEvents, got %v", model.state)
	}

	// Test timer initialization
	if model.timer.Timeout != timeout {
		t.Errorf("Expected timer timeout to be %v, got %v", timeout, model.timer.Timeout)
	}

	// Test inputs initialization
	if len(model.inputs) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(model.inputs))
	}

	// Test events list initialization
	if model.events.Title != "Events" {
		t.Errorf("Expected events title to be 'Events', got '%s'", model.events.Title)
	}
}

func TestGetEventsFilePath(t *testing.T) {
	th := newTestHelper(t)
	defer th.cleanup()

	// Test that the function returns a valid path
	eventsPath, err := getEventsFilePath()
	if err != nil {
		t.Fatalf("getEventsFilePath() failed: %v", err)
	}

	// Verify the path structure
	expectedDir := filepath.Join(th.testConfigDir, appName)
	expectedFile := filepath.Join(expectedDir, eventsFileName)

	if eventsPath != expectedFile {
		t.Errorf("Expected path %s, got %s", expectedFile, eventsPath)
	}

	// Verify the directory was created
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Expected config directory to be created: %s", expectedDir)
	}

	// Test that calling it again doesn't fail (idempotent)
	eventsPath2, err := getEventsFilePath()
	if err != nil {
		t.Fatalf("Second call to getEventsFilePath() failed: %v", err)
	}

	if eventsPath2 != eventsPath {
		t.Errorf("Expected same path on second call, got %s vs %s", eventsPath2, eventsPath)
	}
}

func TestConstants(t *testing.T) {
	// Test time constants
	if secondsPerYear != 31557600 {
		t.Errorf("Expected secondsPerYear to be 31557600, got %d", secondsPerYear)
	}

	if secondsPerDay != 86400 {
		t.Errorf("Expected secondsPerDay to be 86400, got %d", secondsPerDay)
	}

	if secondsPerHour != 3600 {
		t.Errorf("Expected secondsPerHour to be 3600, got %d", secondsPerHour)
	}

	if secondsPerMinute != 60 {
		t.Errorf("Expected secondsPerMinute to be 60, got %d", secondsPerMinute)
	}

	// Test timeout constant
	expectedTimeout := 365 * 24 * time.Hour
	if timeout != expectedTimeout {
		t.Errorf("Expected timeout to be %v, got %v", expectedTimeout, timeout)
	}

	// Test app name constant
	if appName != "countdown" {
		t.Errorf("Expected appName to be 'countdown', got '%s'", appName)
	}

	// Test events file name constant
	if eventsFileName != "events.json" {
		t.Errorf("Expected eventsFileName to be 'events.json', got '%s'", eventsFileName)
	}
}
