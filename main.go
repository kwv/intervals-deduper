package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

var Version = "dev"

func main() {
	dryRun := flag.Bool("dry-run", false, "Preview deletions without making changes")
	interactive := flag.Bool("interactive", false, "Confirm each deletion manually")
	days := flag.Int("days", 0, "Number of days to sync (overrides config)")
	startStr := flag.String("start", "", "Start date (YYYY-MM-DD)")
	endStr := flag.String("end", "", "End date (YYYY-MM-DD)")
	verbose := flag.Bool("verbose", false, "Show all scanned activities")
	dump := flag.String("dump", "", "Export all activities to a JSON file (e.g., dump.json)")
	versionFlag := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("intervals-deduper version %s\n", Version)
		return
	}

	config, err := LoadConfig("config.yml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	var newest, oldest time.Time
	if *startStr != "" {
		oldest, err = time.Parse("2006-01-02", *startStr)
		if err != nil {
			log.Fatalf("Invalid start date: %v", err)
		}
		if *endStr != "" {
			newest, err = time.Parse("2006-01-02", *endStr)
			if err != nil {
				log.Fatalf("Invalid end date: %v", err)
			}
			// Include the full day for the end date
			newest = newest.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		} else {
			newest = time.Now()
		}
	} else {
		if *days > 0 {
			config.DaysToSync = *days
		} else if config.DaysToSync == 0 {
			config.DaysToSync = 30 // Sane default
		}
		newest = time.Now()
		oldest = newest.AddDate(0, 0, -config.DaysToSync)
	}

	client := NewIntervalsClient(config.APIKey, config.AthleteID)
	scoring := NewScoringEngine(config)

	fmt.Printf("🔍 Scanning for duplicates from %s to %s...\n", oldest.Format("2006-01-02"), newest.Format("2006-01-02"))

	activities, err := client.ListActivities(oldest, newest)
	if err != nil {
		log.Fatalf("Error fetching activities: %v", err)
	}

	if *dump != "" {
		fmt.Printf("📦 Fetching details for %d activities and saving to %s...\n", len(activities), *dump)
		var allDetails []ActivityDetail
		for i, a := range activities {
			fmt.Printf("\r   [%d/%d] Fetching %s...", i+1, len(activities), a.ID)
			detail, err := client.GetActivityDetail(a.ID)
			if err != nil {
				fmt.Printf("\n  ⚠️ Failed to fetch details for %s: %v\n", a.ID, err)
				continue
			}
			allDetails = append(allDetails, *detail)
		}
		fmt.Printf("\n💾 Writing to %s...\n", *dump)

		data, err := json.MarshalIndent(allDetails, "", "  ")
		if err != nil {
			log.Fatalf("Error marshaling data: %v", err)
		}

		if err := os.WriteFile(*dump, data, 0644); err != nil {
			log.Fatalf("Error writing file: %v", err)
		}
		fmt.Println("✅ Done!")
		return
	}

	if *verbose {
		fmt.Printf("📊 Scanned %d total activities\n", len(activities))
		for _, a := range activities {
			fmt.Printf("   - [%s] %s (%s)\n", a.ID, a.Name, a.StartDateLocal.Time.Format("2006-01-02 15:04:05"))
		}
	}

	// Sort activities by StartDateLocal
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].StartDateLocal.Time.Before(activities[j].StartDateLocal.Time)
	})

	// Group activities that are within 30 seconds of each other
	var groups [][]Activity
	if len(activities) > 0 {
		currentGroup := []Activity{activities[0]}
		for i := 1; i < len(activities); i++ {
			diff := activities[i].StartDateLocal.Time.Sub(currentGroup[0].StartDateLocal.Time)
			if diff < 0 {
				diff = -diff
			}

			if diff <= 30*time.Second {
				currentGroup = append(currentGroup, activities[i])
			} else {
				if len(currentGroup) > 1 {
					groups = append(groups, currentGroup)
				}
				currentGroup = []Activity{activities[i]}
			}
		}
		if len(currentGroup) > 1 {
			groups = append(groups, currentGroup)
		}
	}

	for _, group := range groups {
		first := group[0]
		fmt.Printf("\n🚩 Found %d suspected duplicates starting around: %s\n", len(group), first.StartDateLocal.Time.Format("2006-01-02 15:04:05"))

		// Fetch details for each to get stream info
		var details []ActivityDetail
		for _, a := range group {
			detail, err := client.GetActivityDetail(a.ID)
			if err != nil {
				fmt.Printf("  ⚠️ Failed to fetch details for %s: %v\n", a.ID, err)
				continue
			}
			details = append(details, *detail)
		}

		if len(details) <= 1 {
			continue
		}

		// Score each activity
		type evaluatedActivity struct {
			Detail ActivityDetail
			Score  Scorecard
		}
		var evaluated []evaluatedActivity
		for _, d := range details {
			evaluated = append(evaluated, evaluatedActivity{
				Detail: d,
				Score:  scoring.Score(&d),
			})
		}

		// Sort by Score DESC, then Updated timestamp DESC, then Created timestamp DESC
		sort.Slice(evaluated, func(i, j int) bool {
			if evaluated[i].Score.Total != evaluated[j].Score.Total {
				return evaluated[i].Score.Total > evaluated[j].Score.Total
			}
			if !evaluated[i].Detail.Updated.Equal(evaluated[j].Detail.Updated.Time) {
				return evaluated[i].Detail.Updated.After(evaluated[j].Detail.Updated.Time)
			}
			return evaluated[i].Detail.CreatedAt.After(evaluated[j].Detail.CreatedAt.Time)
		})

		winner := evaluated[0]
		losers := evaluated[1:]

		winnerSystem := winner.Detail.DeviceName
		winnerSrc := winner.Detail.Source
		if winnerSrc == "OAUTH_CLIENT" && winner.Detail.OAuthClientName != "" {
			winnerSrc = winner.Detail.OAuthClientName
		}
		if winnerSrc != "" && winnerSrc != "OAUTH_CLIENT" && !strings.Contains(strings.ToLower(winner.Detail.DeviceName), strings.ToLower(winnerSrc)) {
			winnerSystem = fmt.Sprintf("%s / %s", winner.Detail.DeviceName, winnerSrc)
		}

		fmt.Printf("  🏆 Winner: [%s] (ID: %s, Score: %.2f) - %s (%s, %s)\n",
			winnerSystem, winner.Detail.ID, winner.Score.Total, winner.Detail.Name,
			formatDistance(winner.Detail.Distance), formatDuration(winner.Detail.MovingTime))
		for _, r := range winner.Score.Reasonings {
			fmt.Printf("    - %s\n", r)
		}

		// --- Name Adoption Logic ---
		if scoring.IsGenericName(winner.Detail.Name, winner.Detail.Type) {
			var candidateNames []string
			for _, loser := range losers {
				candidateNames = append(candidateNames, loser.Detail.Name)
			}
			bestName := scoring.RankCandidateNames(candidateNames, winner.Detail.Type)

			if bestName != "" {
				adoptConfirmed := !*interactive
				if *interactive {
					fmt.Printf("    Adopt descriptive name \"%s\" for %s? [Y/n]: ", bestName, winner.Detail.ID)
					reader := bufio.NewReader(os.Stdin)
					response, _ := reader.ReadString('\n')
					if strings.ToLower(strings.TrimSpace(response)) == "n" {
						adoptConfirmed = false
					}
				}

				if adoptConfirmed {
					if *dryRun {
						fmt.Printf("    [DRY RUN] Would adopt name \"%s\" for %s\n", bestName, winner.Detail.ID)
					} else {
						fmt.Printf("    Adopting name \"%s\"...\n", bestName)
						updates := map[string]interface{}{"name": bestName}
						if err := client.UpdateActivity(winner.Detail.ID, updates); err != nil {
							fmt.Printf("    ❌ Error updating name: %v\n", err)
						} else {
							fmt.Printf("    ✅ Name updated\n")
						}
					}
				}
			}
		}

		// --- Metadata Adoption Logic (Feel, RPE, Description) ---
		metaUpdates := make(map[string]interface{})
		metaReasons := []string{}

		for _, loser := range losers {
			// Migrate Feel
			if winner.Detail.Feel == 0 && loser.Detail.Feel > 0 {
				metaUpdates["feel"] = loser.Detail.Feel
				metaReasons = append(metaReasons, fmt.Sprintf("Feel: %d", loser.Detail.Feel))
				winner.Detail.Feel = loser.Detail.Feel // Clear so we don't pick it up again
			}
			// Migrate RPE
			if winner.Detail.RPE == 0 && loser.Detail.RPE > 0 {
				metaUpdates["icu_rpe"] = loser.Detail.RPE
				metaReasons = append(metaReasons, fmt.Sprintf("RPE: %d", loser.Detail.RPE))
				winner.Detail.RPE = loser.Detail.RPE
			}
			// Migrate Description
			if strings.TrimSpace(winner.Detail.Description) == "" && strings.TrimSpace(loser.Detail.Description) != "" {
				desc := strings.TrimSpace(loser.Detail.Description)
				metaUpdates["description"] = desc
				metaReasons = append(metaReasons, "Description")
				winner.Detail.Description = desc
			}
		}

		if len(metaUpdates) > 0 {
			migrateConfirmed := !*interactive
			msg := strings.Join(metaReasons, ", ")
			if *interactive {
				fmt.Printf("    Adopt metadata (%s) for %s? [Y/n]: ", msg, winner.Detail.ID)
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(response)) == "n" {
					migrateConfirmed = false
				}
			}

			if migrateConfirmed {
				if *dryRun {
					fmt.Printf("    [DRY RUN] Would adopt metadata (%s) for %s\n", msg, winner.Detail.ID)
				} else {
					fmt.Printf("    Adopting metadata (%s)...\n", msg)
					if err := client.UpdateActivity(winner.Detail.ID, metaUpdates); err != nil {
						fmt.Printf("    ❌ Error updating metadata: %v\n", err)
					} else {
						fmt.Printf("    ✅ Metadata updated\n")
					}
				}
			}
		}

		for _, loser := range losers {
			loserSystem := loser.Detail.DeviceName
			loserSrc := loser.Detail.Source
			if loserSrc == "OAUTH_CLIENT" && loser.Detail.OAuthClientName != "" {
				loserSrc = loser.Detail.OAuthClientName
			}
			if loserSrc != "" && loserSrc != "OAUTH_CLIENT" && !strings.Contains(strings.ToLower(loser.Detail.DeviceName), strings.ToLower(loserSrc)) {
				loserSystem = fmt.Sprintf("%s / %s", loser.Detail.DeviceName, loserSrc)
			}
			distDiff := math.Abs(winner.Detail.Distance-loser.Detail.Distance) / math.Max(winner.Detail.Distance, 1.0)
			timeDiff := math.Abs(float64(winner.Detail.MovingTime-loser.Detail.MovingTime)) / math.Max(float64(winner.Detail.MovingTime), 1.0)

			isMismatch := distDiff > 0.5 || timeDiff > 0.25

			warnings := ""
			if distDiff > 0.5 {
				warnings += " ⚠️ [DIST MISMATCH]"
			}
			if timeDiff > 0.25 {
				warnings += " ⚠️ [TIME MISMATCH]"
			}

			if isMismatch {
				fmt.Printf("  ⚠️  Mismatch: [%s] (ID: %s, Score: %.2f) - %s (%s, %s)%s\n",
					loserSystem, loser.Detail.ID, loser.Score.Total, loser.Detail.Name,
					formatDistance(loser.Detail.Distance), formatDuration(loser.Detail.MovingTime), warnings)
				fmt.Printf("    ⏭️  Skipping deletion recommendation for %s due to size difference.\n", loser.Detail.ID)
				continue
			}

			fmt.Printf("  🗑️  To Delete: [%s] (ID: %s, Score: %.2f) - %s (%s, %s)\n",
				loserSystem, loser.Detail.ID, loser.Score.Total, loser.Detail.Name,
				formatDistance(loser.Detail.Distance), formatDuration(loser.Detail.MovingTime))

			for _, r := range loser.Score.Reasonings {
				fmt.Printf("    - %s\n", r)
			}

			deleteConfirmed := !*interactive
			if *interactive {
				fmt.Printf("    Confirm deletion of %s? [y/N]: ", loser.Detail.ID)
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(response)) == "y" {
					deleteConfirmed = true
				}
			}

			if deleteConfirmed {
				if *dryRun {
					fmt.Printf("    [DRY RUN] Would delete %s\n", loser.Detail.ID)
				} else {
					fmt.Printf("    Deleting %s...\n", loser.Detail.ID)
					if err := client.DeleteActivity(loser.Detail.ID); err != nil {
						fmt.Printf("    ❌ Error deleting %s: %v\n", loser.Detail.ID, err)
					} else {
						fmt.Printf("    ✅ Deleted %s\n", loser.Detail.ID)
					}
				}
			} else {
				fmt.Printf("    ⏭️  Skipped deletion of %s\n", loser.Detail.ID)
			}
		}
	}
}
func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	return fmt.Sprintf("%dm %02ds", m, s)
}

func formatDistance(meters float64) string {
	return fmt.Sprintf("%.1fkm", meters/1000.0)
}
