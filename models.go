package main

import (
	"fmt"
	"strings"
	"time"
)

// Config represents the application configuration
type Config struct {
	APIKey            string             `yaml:"api_key"`
	AthleteID         string             `yaml:"athlete_id"`
	Weights           Weights            `yaml:"weights"`
	DevicePriority    []string           `yaml:"device_priority"`
	UploaderPenalties map[string]float64 `yaml:"uploader_penalties"`
	DaysToSync        int                `yaml:"days_to_sync"`
}

// Weights represents the importance of different metrics for heuristic scoring
type Weights struct {
	GPS          float64 `yaml:"gps"`
	HeartRate    float64 `yaml:"heartrate"`
	Power        float64 `yaml:"power"`
	Cadence      float64 `yaml:"cadence"`
	SamplingRate float64 `yaml:"sampling_rate"`
	RPE          float64 `yaml:"rpe"`         // Bonus for presence of RPE/Feel
	Manual       float64 `yaml:"manual"`      // Bonus for notes/description
	CustomName   float64 `yaml:"custom_name"` // Bonus for non-generic names
}

// IntervalsTime handles parsing of ISO-8601 timestamps that may or may not have timezone offsets
type IntervalsTime struct {
	time.Time
}

func (it *IntervalsTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		return nil
	}

	// Try RFC3339 first
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		it.Time = t
		return nil
	}

	// Try ISO-8601 without offset
	t, err = time.Parse("2006-01-02T15:04:05", s)
	if err == nil {
		it.Time = t
		return nil
	}

	return fmt.Errorf("could not parse time %s", s)
}

// Activity represents the summary data retrieved from Intervals.icu
type Activity struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Type                string        `json:"type"`
	StartDateLocal      IntervalsTime `json:"start_date_local"`
	CreatedAt           IntervalsTime `json:"created"`
	DeviceName          string        `json:"device_name"`
	Source              string        `json:"source"`
	IcuRecordingSeconds int           `json:"icu_recording_seconds"`
	Distance            float64       `json:"distance"`
	MovingTime          int           `json:"moving_time"`
	AverageHeartrate    float64       `json:"average_heartrate"`
	AverageWatts        float64       `json:"average_power"`
	HasGPS              bool          `json:"has_gps"` // Derived or checked via streams
	RPE                 int           `json:"icu_rpe"`
	Feel                int           `json:"feel"`
	Description         string        `json:"description"`
	Updated             IntervalsTime `json:"updated"`
	OAuthClientID       int           `json:"oauth_client_id"`
	OAuthClientName     string        `json:"oauth_client_name"`
	PowerMeter          string        `json:"power_meter"`
	PowerMeterSerial    string        `json:"power_meter_serial"`
	PowerMeterBattery   string        `json:"power_meter_battery"`
}

// ActivityDetail provides more in-depth info used for heuristic evaluation
type ActivityDetail struct {
	Activity
	StreamTypes []string `json:"stream_types"`
}

// Scorecard records the breakdown of how an activity was evaluated
type Scorecard struct {
	Total      float64
	Breakdown  map[string]float64
	Reasonings []string
}
