package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestIntervalsTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		// RFC3339
		{`"2023-10-12T14:15:04Z"`, time.Date(2023, 10, 12, 14, 15, 4, 0, time.UTC)},
		// ISO-8601 without offset
		{`"2023-10-12T14:15:04"`, time.Date(2023, 10, 12, 14, 15, 4, 0, time.UTC)},
	}

	for _, tt := range tests {
		var it IntervalsTime
		err := json.Unmarshal([]byte(tt.input), &it)
		if err != nil {
			t.Errorf("UnmarshalJSON(%s) error: %v", tt.input, err)
			continue
		}
		if !it.Equal(tt.want) {
			t.Errorf("UnmarshalJSON(%s) = %v; want %v", tt.input, it.Time, tt.want)
		}
	}
}
