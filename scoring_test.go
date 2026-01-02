package main

import (
	"testing"
)

func TestIsGenericName(t *testing.T) {
	s := &ScoringEngine{}

	tests := []struct {
		name  string
		aType string
		want  bool
	}{
		{"Morning Ride", "Cycling", true},
		{"Morning VirtualRide", "VirtualRide", true},
		{"Lunch Walk", "Walk", true},
		{"Afternoon Rowing", "Rowing", true},
		{"Evening Hike", "Hike", true},
		{"Cycling", "Cycling", true},
		{"Workout", "Cycling", true},
		{"Fox Creek Trails", "Cycling", false},
		{"Morning GravelRide", "Cycling", false}, // "GravelRide" != "Cycling"
		{"Ellisville - Weldon", "Cycling", false},
		{"Untitled", "Cycling", true},
		{"", "Cycling", true},
	}

	for _, tt := range tests {
		if got := s.IsGenericName(tt.name, tt.aType); got != tt.want {
			t.Errorf("IsGenericName(%q, %q) = %v; want %v", tt.name, tt.aType, got, tt.want)
		}
	}
}

func TestRankCandidateNames(t *testing.T) {
	s := &ScoringEngine{}

	tests := []struct {
		names []string
		aType string
		want  string
	}{
		{[]string{"Morning Ride", "Ellisville - Weldon", "Cycling"}, "Cycling", "Ellisville - Weldon"},
		{[]string{"Morning Ride", "Morning GravelRide"}, "Cycling", "Morning GravelRide"},
		{[]string{"Lunch Walk", "Centaur - Pond"}, "Walk", "Centaur - Pond"},
	}

	for _, tt := range tests {
		if got := s.RankCandidateNames(tt.names, tt.aType); got != tt.want {
			t.Errorf("RankCandidateNames(%v, %q) = %q; want %q", tt.names, tt.aType, got, tt.want)
		}
	}
}

func TestScoring(t *testing.T) {
	config := Config{
		Weights: Weights{
			GPS:        10,
			Power:      5,
			CustomName: 2,
		},
		DevicePriority: []string{"Wahoo"},
	}
	s := NewScoringEngine(&config)

	// High quality activity
	detail1 := &ActivityDetail{
		Activity: Activity{
			Name:       "Epic Ride",
			Type:       "Cycling",
			DeviceName: "Wahoo ELEMNT",
		},
		StreamTypes: []string{"latlng", "watts"},
	}
	score1 := s.Score(detail1)

	if score1.Total <= 15 {
		t.Errorf("Expected score > 15 for high quality activity, got %.2f", score1.Total)
	}

	// Low quality activity
	detail2 := &ActivityDetail{
		Activity: Activity{
			Name:       "Cycling",
			Type:       "Cycling",
			DeviceName: "Phone",
		},
		StreamTypes: []string{"time"},
	}
	score2 := s.Score(detail2)

	if score2.Total >= score1.Total {
		t.Errorf("Expected low quality score (%.2f) to be less than high quality (%.2f)", score2.Total, score1.Total)
	}
}
