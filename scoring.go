package main

import (
	"fmt"
	"math"
	"strings"
)

type ScoringEngine struct {
	Config *Config
}

func NewScoringEngine(config *Config) *ScoringEngine {
	return &ScoringEngine{
		Config: config,
	}
}

func (s *ScoringEngine) IsGenericName(name string, activityType string) bool {
	if name == "" || strings.ToLower(name) == "untitled" {
		return true
	}

	trimmed := strings.ToLower(strings.TrimSpace(name))
	lowType := strings.ToLower(activityType)

	// Direct match to type (e.g., "Cycling", "Run")
	if trimmed == lowType || trimmed == "cycling" || trimmed == "ride" || trimmed == "workout" {
		return true
	}

	// Pattern match: "{Time} {Type}"
	times := []string{"morning", "afternoon", "evening", "night", "lunch"}
	for _, t := range times {
		// e.g., "morning ride", "lunch walk"
		if trimmed == t+" "+lowType {
			return true
		}
		// Also catch common generic mappings (e.g. Type=VirtualRide but name="Morning Ride")
		if trimmed == t+" ride" || trimmed == t+" workout" {
			return true
		}
	}

	return false
}

func (s *ScoringEngine) RankCandidateNames(names []string, activityType string) string {
	var bestName string
	bestScore := -999.0

	for _, name := range names {
		if s.IsGenericName(name, activityType) {
			continue
		}

		score := 1.0 // Base score for non-generic names

		// Pattern: "Location - Landmark" or similar rich patterns
		if strings.Contains(name, " - ") {
			score += 100.0
		}

		// Length bonus (bias towards descriptiveness)
		score += float64(len(name)) * 0.1

		// Penalty for starts with "Morning", "Afternoon" etc if it's just two words
		// (e.g. "Morning Ride" is generic, but "Morning GravelRide" is slightly less so)
		keywords := []string{"morning", "afternoon", "evening", "night", "lunch"}
		lower := strings.ToLower(name)
		for _, k := range keywords {
			if strings.HasPrefix(lower, k) {
				score -= 10.0
				break
			}
		}

		if score > bestScore {
			bestScore = score
			bestName = name
		}
	}

	return bestName
}

func (s *ScoringEngine) Score(detail *ActivityDetail) Scorecard {
	card := Scorecard{
		Breakdown:  make(map[string]float64),
		Reasonings: []string{},
	}

	// 1. Stream Density Scoring
	s.evaluateStreams(detail, &card)

	// 2. Sampling Frequency Scoring
	s.evaluateSampling(detail, &card)

	// 3. Device Priority Scoring
	s.evaluateDevice(detail, &card)

	// 4. User Interaction Metrics
	s.evaluateInteractions(detail, &card)

	// 5. Name Analysis
	s.evaluateName(detail, &card)

	// 6. Uploader Penalties
	s.evaluateUploader(detail, &card)

	// Calculate Total
	for _, score := range card.Breakdown {
		card.Total += score
	}

	return card
}

func (s *ScoringEngine) evaluateStreams(detail *ActivityDetail, card *Scorecard) {
	hasWatts := false
	hasHR := false
	hasGPS := false
	hasCadence := false

	for _, t := range detail.StreamTypes {
		switch t {
		case "watts":
			hasWatts = true
		case "heartrate":
			hasHR = true
		case "latlng":
			hasGPS = true
		case "cadence":
			hasCadence = true
		}
	}

	if hasWatts {
		score := s.Config.Weights.Power
		card.Breakdown["Power Stream"] = score
		card.Reasonings = append(card.Reasonings, "Contains power/watts data.")
	}
	if hasHR {
		score := s.Config.Weights.HeartRate
		card.Breakdown["HeartRate Stream"] = score
		card.Reasonings = append(card.Reasonings, "Contains heart rate data.")
	}
	if hasGPS {
		score := s.Config.Weights.GPS
		card.Breakdown["GPS/Map Stream"] = score
		card.Reasonings = append(card.Reasonings, "Contains GPS/latlng map data.")
	}
	if hasCadence {
		score := s.Config.Weights.Cadence
		card.Breakdown["Cadence Stream"] = score
		card.Reasonings = append(card.Reasonings, "Contains cadence data.")
	}
}

func (s *ScoringEngine) evaluateSampling(detail *ActivityDetail, card *Scorecard) {
	if detail.IcuRecordingSeconds <= 0 || detail.MovingTime <= 0 {
		return
	}

	// Sampling rate: Points per second of moving time.
	// 1.0 means 1s recording. Higher is better (unlikely in raw data but theoretically possible).
	rate := float64(detail.IcuRecordingSeconds) / float64(detail.MovingTime)

	// Normalize: we cap it at 1.0 (expected max density)
	normalizedRate := math.Min(rate, 1.0)

	score := normalizedRate * s.Config.Weights.SamplingRate
	card.Breakdown["Sampling Density"] = score
	card.Reasonings = append(card.Reasonings, fmt.Sprintf("Sampling density: %.0f%% of moving time recorded.", normalizedRate*100))
}

func (s *ScoringEngine) evaluateDevice(detail *ActivityDetail, card *Scorecard) {
	// Check recording device, source uploader, and any specific power meter sensors
	searchString := strings.ToLower(fmt.Sprintf("%s %s %s %s", detail.DeviceName, detail.Source, detail.PowerMeter, detail.OAuthClientName))

	for i, preferred := range s.Config.DevicePriority {
		if strings.Contains(searchString, strings.ToLower(preferred)) {
			// Higher bonus for higher items in the list
			bonus := float64(len(s.Config.DevicePriority)-i) * 2.0
			card.Breakdown["Device/Sensor Priority: "+preferred] = bonus
			card.Reasonings = append(card.Reasonings, fmt.Sprintf("Matches preferred device/sensor: %s", preferred))
			break
		}
	}
}

func (s *ScoringEngine) evaluateInteractions(detail *ActivityDetail, card *Scorecard) {
	if detail.RPE > 0 {
		score := s.Config.Weights.RPE
		card.Breakdown["RPE/Feel"] = score
		card.Reasonings = append(card.Reasonings, fmt.Sprintf("User provided RPE: %d", detail.RPE))
	} else if detail.Feel > 0 {
		score := s.Config.Weights.RPE
		card.Breakdown["RPE/Feel"] = score
		card.Reasonings = append(card.Reasonings, fmt.Sprintf("User provided Feel: %d", detail.Feel))
	}

	if strings.TrimSpace(detail.Description) != "" {
		score := s.Config.Weights.Manual
		card.Breakdown["Manual Description"] = score
		card.Reasonings = append(card.Reasonings, "Activity has custom notes/description.")
	}
}

func (s *ScoringEngine) evaluateName(detail *ActivityDetail, card *Scorecard) {
	if !s.IsGenericName(detail.Name, detail.Type) {
		// Only award custom name bonus if not from a penalized uploader (like RunGap)
		// Only award custom name bonus if not from a penalized uploader (like RunGap)
		// as those tools often set automated titles.
		isPenalized := false
		uploader := strings.ToLower(detail.Source)
		if detail.Source == "OAUTH_CLIENT" {
			uploader = strings.ToLower(detail.OAuthClientName)
		}

		for key := range s.Config.UploaderPenalties {
			if strings.Contains(uploader, strings.ToLower(key)) {
				isPenalized = true
				break
			}
		}

		if !isPenalized {
			score := s.Config.Weights.CustomName
			card.Breakdown["Custom Name"] = score
			card.Reasonings = append(card.Reasonings, fmt.Sprintf("Appears to have a custom name: %s", detail.Name))
		}
	}
}

func (s *ScoringEngine) evaluateUploader(detail *ActivityDetail, card *Scorecard) {
	uploader := detail.Source
	if uploader == "OAUTH_CLIENT" && detail.OAuthClientName != "" {
		uploader = detail.OAuthClientName
	}

	for key, penalty := range s.Config.UploaderPenalties {
		if strings.Contains(strings.ToLower(uploader), strings.ToLower(key)) {
			card.Breakdown["Uploader Penalty: "+key] = -penalty
			card.Reasonings = append(card.Reasonings, fmt.Sprintf("Penalized for using indirect sync tool: %s", uploader))
		}
	}
}
