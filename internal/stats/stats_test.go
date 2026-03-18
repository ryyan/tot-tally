package stats

import (
	"testing"
	"time"
	totConfig "tot-tally/internal/config"
	totModels "tot-tally/internal/models"
)

func TestFormatAvgGap(t *testing.T) {
	e := NewEngine(&totConfig.Config{})

	tests := []struct {
		name     string
		times    []*time.Time
		expected string
	}{
		{
			name:     "Empty slice",
			times:    []*time.Time{},
			expected: "---",
		},
		{
			name:     "One element",
			times:    []*time.Time{{}},
			expected: "---",
		},
		{
			name: "60 minute gap",
			times: []*time.Time{
				nil,
				nil,
			},
			expected: "1h 0m",
		},
	}

	// Update times for "60 minute gap" to be exact
	now := time.Now()
	t60m := now.Add(-60 * time.Minute)
	tests[2].times = []*time.Time{&now, &t60m}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.FormatAvgGap(tt.times)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRecalculateStats(t *testing.T) {
	e := NewEngine(&totConfig.Config{})
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)

	tot := &totModels.Tot{
		Tallies: []totModels.Tally{
			{Kind: "🍼100", Time: &earlier},
			{Kind: totConfig.TallyKindMap[9], Time: &earlier},  // Snack
			{Kind: totConfig.TallyKindMap[10], Time: &earlier}, // Meal
			{Kind: totConfig.TallyKindMap[11], Time: &earlier}, // Pee
			{Kind: totConfig.TallyKindMap[12], Time: &earlier}, // Poo
			{Kind: totConfig.TallyKindMap[14], Time: &earlier}, // Bath
			{Kind: totConfig.TallyKindMap[15], Time: &earlier}, // Brush
			{Kind: totConfig.TallyKindMap[16], Time: &earlier}, // Nurse L
			{Kind: totConfig.TallyKindMap[17], Time: &earlier}, // Nurse R
		},
	}

	e.RecalculateStats(tot)

	if tot.Stats.LastMilk == nil || !tot.Stats.LastMilk.Equal(earlier) {
		t.Error("LastMilk mismatch")
	}
	if tot.Stats.LastSnack == nil || !tot.Stats.LastSnack.Equal(earlier) {
		t.Error("LastSnack mismatch")
	}
	if tot.Stats.LastPee == nil || !tot.Stats.LastPee.Equal(earlier) {
		t.Error("LastPee mismatch")
	}
	if tot.Stats.LastNurse == nil || !tot.Stats.LastNurse.Equal(earlier) {
		t.Error("LastNurse mismatch")
	}
}

func TestGenerateStats_Detailed(t *testing.T) {
	cfg := &totConfig.Config{MaxTallies: 100}
	e := NewEngine(cfg)
	tz := time.UTC
	now := time.Date(2023, 10, 27, 12, 0, 0, 0, tz)

	t0 := now
	t12h := now.Add(-11 * time.Hour) // within 12h
	t1d := now.Add(-23 * time.Hour)  // within 24h, not 12h

	tot := &totModels.Tot{
		Tallies: []totModels.Tally{
			{Kind: "🤱L", Time: &t0},
			{Kind: totConfig.TallyKindMap[11], Time: &t12h}, // Pee
			{Kind: totConfig.TallyKindMap[12], Time: &t1d},  // Poo
		},
	}

	s, err := e.GenerateStats(tot, tz, now)
	if err != nil {
		t.Fatalf("GenerateStats failed: %v", err)
	}

	if s.Last12HoursNurse != "1" {
		t.Errorf("Expected 12h nurse 1, got %s", s.Last12HoursNurse)
	}
	if s.Last12HoursPee != "1" {
		t.Errorf("Expected 12h pee 1, got %s", s.Last12HoursPee)
	}
	if s.Last24HoursPoo != "1" {
		t.Errorf("Expected 24h poo 1, got %s", s.Last24HoursPoo)
	}
	if s.Last12HoursPoo != "0" {
		t.Errorf("Expected 12h poo 0, got %s", s.Last12HoursPoo)
	}
}

func TestGenerateStats(t *testing.T) {
	cfg := &totConfig.Config{MaxTallies: 100}
	e := NewEngine(cfg)
	tz := time.UTC
	now := time.Date(2023, 10, 27, 12, 0, 0, 0, tz)

	t0 := now
	t1 := now.AddDate(0, 0, -1)
	t2 := now.AddDate(0, 0, -2)
	t3 := now.AddDate(0, 0, -3)
	t4 := now.AddDate(0, 0, -4).Add(-1 * time.Hour)

	tot := &totModels.Tot{
		Tallies: []totModels.Tally{
			{Kind: "🍼100", Time: &t0},
			{Kind: "🍼120", Time: &t1},
			{Kind: "🍼80", Time: &t2},
			{Kind: "🍼90", Time: &t3},
			{Kind: "🍼0", Time: &t4},
		},
	}

	s, err := e.GenerateStats(tot, tz, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.TodayMilk != "100" || s.YesterdayMilk != "120" || s.TwoDaysAgoMilk != "80" || s.ThreeDaysAgoMilk != "90" {
		t.Errorf("Daily milk buckets mismatch: Today=%s, Yesterday=%s, 2d=%s, 3d=%s", s.TodayMilk, s.YesterdayMilk, s.TwoDaysAgoMilk, s.ThreeDaysAgoMilk)
	}
}

func TestGenerateStats_MalformedMilk(t *testing.T) {
	e := NewEngine(&totConfig.Config{})
	tz := time.UTC
	now := time.Date(2023, 10, 27, 12, 0, 0, 0, tz)

	tot := &totModels.Tot{
		Tallies: []totModels.Tally{
			{Kind: "🍼not-a-number", Time: &now},
		},
	}

	s, err := e.GenerateStats(tot, tz, now)
	if err == nil {
		t.Fatal("Expected GenerateStats to fail for malformed milk, got nil")
	}

	if s.TodayMilk != "" { // Now it returns empty struct on error
		t.Errorf("Expected empty TodayMilk for malformed milk, got %s", s.TodayMilk)
	}
}

func TestGenerateStats_AllBranches(t *testing.T) {
	cfg := &totConfig.Config{MaxTallies: 100}
	e := NewEngine(cfg)
	tz := time.UTC
	now := time.Date(2023, 10, 27, 12, 0, 0, 0, tz)

	// Helper to create time at specific relative points
	tAt := func(d, h int) *time.Time {
		tm := now.AddDate(0, 0, -d).Add(time.Duration(-h) * time.Hour)
		return &tm
	}

	tot := &totModels.Tot{
		Tallies: []totModels.Tally{
			// Today & Last 12h & Last 24h
			{Kind: "🍼100", Time: tAt(0, 1)},
			{Kind: "🤱L", Time: tAt(0, 2)},
			{Kind: totConfig.TallyKindMap[11], Time: tAt(0, 3)}, // Pee
			{Kind: totConfig.TallyKindMap[12], Time: tAt(0, 4)}, // Poo

			// Today & Last 24h (not last 12h) - if current time is late in the day
			// But easier to just use Yesterday start
			{Kind: "🍼50", Time: tAt(1, 0)}, // Exactly 24h ago if now is start of day

			// Yesterday
			{Kind: "🍼200", Time: tAt(1, 1)},
			{Kind: "🤱R", Time: tAt(1, 2)},
			{Kind: totConfig.TallyKindMap[11], Time: tAt(1, 3)},
			{Kind: totConfig.TallyKindMap[12], Time: tAt(1, 4)},

			// Two Days Ago
			{Kind: "🍼300", Time: tAt(2, 1)},
			{Kind: "🤱L", Time: tAt(2, 2)},
			{Kind: totConfig.TallyKindMap[11], Time: tAt(2, 3)},
			{Kind: totConfig.TallyKindMap[12], Time: tAt(2, 4)},

			// Three Days Ago
			{Kind: "🍼400", Time: tAt(3, 1)},
			{Kind: "🤱R", Time: tAt(3, 2)},
			{Kind: totConfig.TallyKindMap[11], Time: tAt(3, 3)},
			{Kind: totConfig.TallyKindMap[12], Time: tAt(3, 4)},

			// Combo Pee/Poo
			{Kind: totConfig.TallyKindMap[13], Time: tAt(0, 5)},
		},
	}

	s, err := e.GenerateStats(tot, tz, now)
	if err != nil {
		t.Fatalf("GenerateStats failed: %v", err)
	}

	// Verify some key values to ensure branches were hit
	if s.TodayMilk == "0" && len(tot.Tallies) > 0 {
		t.Error("TodayMilk should not be 0")
	}
	if s.YesterdayMilk == "0" {
		t.Error("YesterdayMilk should not be 0")
	}
	if s.TwoDaysAgoMilk == "0" {
		t.Error("TwoDaysAgoMilk should not be 0")
	}
	if s.ThreeDaysAgoMilk == "0" {
		t.Error("ThreeDaysAgoMilk should not be 0")
	}

	// Check combo
	// Today has 1 Pee (tAt(0,3)) + 1 Combo (tAt(0,5)) = 2 Pee
	if s.TodayPee != "2" {
		t.Errorf("Expected TodayPee 2, got %s", s.TodayPee)
	}
	if s.TodayPoo != "2" {
		t.Errorf("Expected TodayPoo 2, got %s", s.TodayPoo)
	}
}

func TestGenerateStats_Exhaustive(t *testing.T) {
	cfg := &totConfig.Config{MaxTallies: 100}
	e := NewEngine(cfg)
	tz := time.UTC
	now := time.Date(2023, 10, 27, 12, 0, 0, 0, tz)

	// Start of each day
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, tz)
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	twoDaysAgoStart := todayStart.AddDate(0, 0, -2)
	threeDaysAgoStart := todayStart.AddDate(0, 0, -3)

	tot := &totModels.Tot{
		Tallies: []totModels.Tally{
			// Today
			{Kind: "🍼100", Time: &todayStart},
			{Kind: "🤱L", Time: &todayStart},
			{Kind: totConfig.TallyKindMap[11], Time: &todayStart},
			{Kind: totConfig.TallyKindMap[12], Time: &todayStart},

			// Yesterday
			{Kind: "🍼110", Time: &yesterdayStart},
			{Kind: "🤱R", Time: &yesterdayStart},
			{Kind: totConfig.TallyKindMap[11], Time: &yesterdayStart},
			{Kind: totConfig.TallyKindMap[12], Time: &yesterdayStart},

			// 2 Days Ago
			{Kind: "🍼120", Time: &twoDaysAgoStart},
			{Kind: "🤱L", Time: &twoDaysAgoStart},
			{Kind: totConfig.TallyKindMap[11], Time: &twoDaysAgoStart},
			{Kind: totConfig.TallyKindMap[12], Time: &twoDaysAgoStart},

			// 3 Days Ago
			{Kind: "🍼130", Time: &threeDaysAgoStart},
			{Kind: "🤱R", Time: &threeDaysAgoStart},
			{Kind: totConfig.TallyKindMap[11], Time: &threeDaysAgoStart},
			{Kind: totConfig.TallyKindMap[12], Time: &threeDaysAgoStart},

			// 4 Days Ago (Buffer)
			{Kind: "🍼0", Time: func() *time.Time { t := threeDaysAgoStart.AddDate(0, 0, -1); return &t }()},
		},
	}

	s, err := e.GenerateStats(tot, tz, now)
	if err != nil {
		t.Fatalf("GenerateStats failed: %v", err)
	}

	// Verify all time buckets and types
	checks := []struct {
		val      string
		expected string
		name     string
	}{
		{s.TodayMilk, "100", "TodayMilk"},
		{s.TodayNurse, "1", "TodayNurse"},
		{s.TodayPee, "1", "TodayPee"},
		{s.TodayPoo, "1", "TodayPoo"},

		{s.YesterdayMilk, "110", "YesterdayMilk"},
		{s.YesterdayNurse, "1", "YesterdayNurse"},
		{s.YesterdayPee, "1", "YesterdayPee"},
		{s.YesterdayPoo, "1", "YesterdayPoo"},

		{s.TwoDaysAgoMilk, "120", "TwoDaysAgoMilk"},
		{s.TwoDaysAgoNurse, "1", "TwoDaysAgoNurse"},
		{s.TwoDaysAgoPee, "1", "TwoDaysAgoPee"},
		{s.TwoDaysAgoPoo, "1", "TwoDaysAgoPoo"},

		{s.ThreeDaysAgoMilk, "130", "ThreeDaysAgoMilk"},
		{s.ThreeDaysAgoNurse, "1", "ThreeDaysAgoNurse"},
		{s.ThreeDaysAgoPee, "1", "ThreeDaysAgoPee"},
		{s.ThreeDaysAgoPoo, "1", "ThreeDaysAgoPoo"},

		// 3-day averages (Sum of Yesterday, 2d ago, 3d ago) / 3
		// Milk: (110 + 120 + 130) / 3 = 120
		// Others: (1 + 1 + 1) / 3 = 1
		{s.ThreeDayAvgMilk, "120", "ThreeDayAvgMilk"},
		{s.ThreeDayAvgNurse, "1", "ThreeDayAvgNurse"},
		{s.ThreeDayAvgPee, "1", "ThreeDayAvgPee"},
		{s.ThreeDayAvgPoo, "1", "ThreeDayAvgPoo"},
	}

	for _, c := range checks {
		if c.val != c.expected {
			t.Errorf("Expected %s to be %s, got %s", c.name, c.expected, c.val)
		}
	}
}

func TestGenerateStats_HistoryBuffer(t *testing.T) {
	cfg := &totConfig.Config{MaxTallies: 100}
	e := NewEngine(cfg)
	tz := time.UTC
	now := time.Date(2023, 10, 27, 12, 0, 0, 0, tz)

	// Case 1: Less than 4 days of history (oldest is 3 days ago)
	t3 := now.AddDate(0, 0, -3)
	totSmall := &totModels.Tot{
		Tallies: []totModels.Tally{
			{Kind: "🍼100", Time: &t3},
		},
	}

	s, err := e.GenerateStats(totSmall, tz, now)
	if err != nil {
		t.Fatalf("GenerateStats failed: %v", err)
	}

	if s.ThreeDayAvgMilk != "---" {
		t.Errorf("Expected 3-day avg to be --- for insufficient history, got %s", s.ThreeDayAvgMilk)
	}
	if s.AvgGapMilk != "---" {
		t.Errorf("Expected avg gap to be --- for insufficient history, got %s", s.AvgGapMilk)
	}

	// Case 2: Enough history (oldest is 4.1 days ago)
	t4 := now.AddDate(0, 0, -4).Add(-1 * time.Hour)
	t1 := now.AddDate(0, 0, -1)
	t2 := now.AddDate(0, 0, -2)
	totLarge := &totModels.Tot{
		Tallies: []totModels.Tally{
			{Kind: "🍼100", Time: &t1},
			{Kind: "🍼100", Time: &t2},
			{Kind: "🍼100", Time: &t4},
		},
	}

	s, err = e.GenerateStats(totLarge, tz, now)
	if err != nil {
		t.Fatalf("GenerateStats failed: %v", err)
	}

	// (100+100+0) / 3 = 66
	if s.ThreeDayAvgMilk == "---" {
		t.Error("Expected 3-day avg to be populated for sufficient history")
	}
	// Gap between t1 and t2 is 24h
	if s.AvgGapMilk == "---" {
		t.Error("Expected avg gap to be populated for sufficient history")
	}
}
