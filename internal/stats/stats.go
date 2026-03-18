// stats.go is the calculation engine for historical trends, averages, and display data.
package stats

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	totConfig "tot-tally/internal/config"
	totModels "tot-tally/internal/models"
)

// Engine handles all mathematical calculations for the dashboard.
type Engine struct {
	config *totConfig.Config
}

// NewEngine initializes the stats calculation logic.
func NewEngine(cfg *totConfig.Config) *Engine {
	return &Engine{config: cfg}
}

// GenerateStats performs the calculation of trends and daily totals.
func (e *Engine) GenerateStats(tot *totModels.Tot, tzLocation *time.Location, now time.Time) (totModels.GeneratedStats, error) {
	now = now.In(tzLocation)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, tzLocation)
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	twoDaysAgoStart := todayStart.AddDate(0, 0, -2)
	threeDaysAgoStart := todayStart.AddDate(0, 0, -3)
	twelveHoursAgo := now.Add(-12 * time.Hour)
	twentyFourHoursAgo := now.Add(-24 * time.Hour)

	// Determine if we have enough history (oldest tally >= 4 days old)
	hasEnoughHistory := false
	if len(tot.Tallies) > 0 {
		oldestTally := tot.Tallies[len(tot.Tallies)-1]
		fourDaysAgo := now.AddDate(0, 0, -4)
		if oldestTally.Time.Before(fourDaysAgo) {
			hasEnoughHistory = true
		}
	}

	var (
		todayMilk, todayNurse, todayPee, todayPoo                             int
		yesterdayMilk, yesterdayNurse, yesterdayPee, yesterdayPoo             int
		twoDaysAgoMilk, twoDaysAgoNurse, twoDaysAgoPee, twoDaysAgoPoo         int
		threeDaysAgoMilk, threeDaysAgoNurse, threeDaysAgoPee, threeDaysAgoPoo int
		last12Milk, last12Nurse, last12Pee, last12Poo                         int
		last24Milk, last24Nurse, last24Pee, last24Poo                         int
		threeDaySumMilk, threeDaySumNurse, threeDaySumPee, threeDaySumPoo     int
	)

	milkTimesGap := make([]*time.Time, 0, e.config.MaxTallies)
	nurseTimesGap := make([]*time.Time, 0, e.config.MaxTallies)
	peeTimesGap := make([]*time.Time, 0, e.config.MaxTallies)
	pooTimesGap := make([]*time.Time, 0, e.config.MaxTallies)

	for i := range tot.Tallies {
		tally := &tot.Tallies[i]
		tLocal := tally.Time.In(tzLocation)

		isLast12 := !tLocal.Before(twelveHoursAgo)
		isLast24 := !tLocal.Before(twentyFourHoursAgo)
		isToday := !tLocal.Before(todayStart)
		isYesterday := tLocal.Before(todayStart) && !tLocal.Before(yesterdayStart)
		isTwoDaysAgo := tLocal.Before(yesterdayStart) && !tLocal.Before(twoDaysAgoStart)
		isThreeDaysAgo := tLocal.Before(twoDaysAgoStart) && !tLocal.Before(threeDaysAgoStart)
		isThreeDayRange := tLocal.Before(todayStart) && !tLocal.Before(threeDaysAgoStart)

		milkAmount := 0
		isMilk := false
		if amountStr, found := strings.CutPrefix(tally.Kind, "🍼"); found {
			isMilk = true
			var err error
			milkAmount, err = strconv.Atoi(amountStr)
			if err != nil {
				return totModels.GeneratedStats{}, fmt.Errorf("stats: invalid milk amount %q: %w", amountStr, err)
			}
			if isThreeDayRange {
				milkTimesGap = append(milkTimesGap, tally.Time)
			}
		}

		isNurse := tally.Kind == "🤱L" || tally.Kind == "🤱R"
		if isNurse && isThreeDayRange {
			nurseTimesGap = append(nurseTimesGap, tally.Time)
		}
		isPee := tally.Kind == totConfig.TallyKindMap[11] || tally.Kind == totConfig.TallyKindMap[13]
		if isPee && isThreeDayRange {
			peeTimesGap = append(peeTimesGap, tally.Time)
		}
		isPoo := tally.Kind == totConfig.TallyKindMap[12] || tally.Kind == totConfig.TallyKindMap[13]
		if isPoo && isThreeDayRange {
			pooTimesGap = append(pooTimesGap, tally.Time)
		}

		if isLast12 {
			if isMilk {
				last12Milk += milkAmount
			}
			if isNurse {
				last12Nurse++
			}
			if isPee {
				last12Pee++
			}
			if isPoo {
				last12Poo++
			}
		}
		if isLast24 {
			if isMilk {
				last24Milk += milkAmount
			}
			if isNurse {
				last24Nurse++
			}
			if isPee {
				last24Pee++
			}
			if isPoo {
				last24Poo++
			}
		}
		if isToday {
			if isMilk {
				todayMilk += milkAmount
			}
			if isNurse {
				todayNurse++
			}
			if isPee {
				todayPee++
			}
			if isPoo {
				todayPoo++
			}
		}
		if isYesterday {
			if isMilk {
				yesterdayMilk += milkAmount
			}
			if isNurse {
				yesterdayNurse++
			}
			if isPee {
				yesterdayPee++
			}
			if isPoo {
				yesterdayPoo++
			}
		}
		if isTwoDaysAgo {
			if isMilk {
				twoDaysAgoMilk += milkAmount
			}
			if isNurse {
				twoDaysAgoNurse++
			}
			if isPee {
				twoDaysAgoPee++
			}
			if isPoo {
				twoDaysAgoPoo++
			}
		}
		if isThreeDaysAgo {
			if isMilk {
				threeDaysAgoMilk += milkAmount
			}
			if isNurse {
				threeDaysAgoNurse++
			}
			if isPee {
				threeDaysAgoPee++
			}
			if isPoo {
				threeDaysAgoPoo++
			}
		}
		if isThreeDayRange {
			if isMilk {
				threeDaySumMilk += milkAmount
			}
			if isNurse {
				threeDaySumNurse++
			}
			if isPee {
				threeDaySumPee++
			}
			if isPoo {
				threeDaySumPoo++
			}
		}
	}

	formatAvg := func(sum int, days int) string {
		return strconv.Itoa(sum / days)
	}

	res := totModels.GeneratedStats{
		Last12HoursMilk: strconv.Itoa(last12Milk), Last12HoursNurse: strconv.Itoa(last12Nurse),
		Last12HoursPee: strconv.Itoa(last12Pee), Last12HoursPoo: strconv.Itoa(last12Poo),
		Last24HoursMilk: strconv.Itoa(last24Milk), Last24HoursNurse: strconv.Itoa(last24Nurse),
		Last24HoursPee: strconv.Itoa(last24Pee), Last24HoursPoo: strconv.Itoa(last24Poo),
		TodayMilk: strconv.Itoa(todayMilk), TodayNurse: strconv.Itoa(todayNurse),
		TodayPee: strconv.Itoa(todayPee), TodayPoo: strconv.Itoa(todayPoo),
		YesterdayMilk: strconv.Itoa(yesterdayMilk), YesterdayNurse: strconv.Itoa(yesterdayNurse),
		YesterdayPee: strconv.Itoa(yesterdayPee), YesterdayPoo: strconv.Itoa(yesterdayPoo),
		TwoDaysAgoMilk: strconv.Itoa(twoDaysAgoMilk), TwoDaysAgoNurse: strconv.Itoa(twoDaysAgoNurse),
		TwoDaysAgoPee: strconv.Itoa(twoDaysAgoPee), TwoDaysAgoPoo: strconv.Itoa(twoDaysAgoPoo),
		ThreeDaysAgoMilk: strconv.Itoa(threeDaysAgoMilk), ThreeDaysAgoNurse: strconv.Itoa(threeDaysAgoNurse),
		ThreeDaysAgoPee: strconv.Itoa(threeDaysAgoPee), ThreeDaysAgoPoo: strconv.Itoa(threeDaysAgoPoo),
		ThreeDayAvgMilk: formatAvg(threeDaySumMilk, 3), ThreeDayAvgNurse: formatAvg(threeDaySumNurse, 3),
		ThreeDayAvgPee: formatAvg(threeDaySumPee, 3), ThreeDayAvgPoo: formatAvg(threeDaySumPoo, 3),
		AvgGapMilk: e.FormatAvgGap(milkTimesGap), AvgGapNurse: e.FormatAvgGap(nurseTimesGap),
		AvgGapPee: e.FormatAvgGap(peeTimesGap), AvgGapPoo: e.FormatAvgGap(pooTimesGap),
	}

	if !hasEnoughHistory {
		res.ThreeDayAvgMilk = "---"
		res.ThreeDayAvgNurse = "---"
		res.ThreeDayAvgPee = "---"
		res.ThreeDayAvgPoo = "---"
		res.AvgGapMilk = "---"
		res.AvgGapNurse = "---"
		res.AvgGapPee = "---"
		res.AvgGapPoo = "---"
	}

	return res, nil
}

// FormatAvgGap calculates the mean time between events.
func (e *Engine) FormatAvgGap(times []*time.Time) string {
	n := len(times)
	if n < 2 {
		return "---"
	}
	totalDuration := times[0].Sub(*times[n-1])
	avgSecs := int64(totalDuration.Seconds()) / int64(n-1)
	avgDur := time.Duration(avgSecs) * time.Second
	hours := int(avgDur.Hours())
	mins := int(avgDur.Minutes()) % 60
	if hours == 0 {
		return fmt.Sprintf("%dm", mins)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// RecalculateStats rebuilds latest activity markers.
func (e *Engine) RecalculateStats(tot *totModels.Tot) {
	tot.Stats = totModels.Stats{}
	for i := range tot.Tallies {
		t := &tot.Tallies[i]
		if strings.HasPrefix(t.Kind, "🍼") {
			if tot.Stats.LastMilk == nil {
				tot.Stats.LastMilk = t.Time
			}
			continue
		}
		switch t.Kind {
		case totConfig.TallyKindMap[9]:
			if tot.Stats.LastSnack == nil {
				tot.Stats.LastSnack = t.Time
			}
		case totConfig.TallyKindMap[10]:
			if tot.Stats.LastMeal == nil {
				tot.Stats.LastMeal = t.Time
			}
		case totConfig.TallyKindMap[11]:
			if tot.Stats.LastPee == nil {
				tot.Stats.LastPee = t.Time
			}
		case totConfig.TallyKindMap[12]:
			if tot.Stats.LastPoo == nil {
				tot.Stats.LastPoo = t.Time
			}
		case totConfig.TallyKindMap[14]:
			if tot.Stats.LastBath == nil {
				tot.Stats.LastBath = t.Time
			}
		case totConfig.TallyKindMap[15]:
			if tot.Stats.LastBrush == nil {
				tot.Stats.LastBrush = t.Time
			}
		case totConfig.TallyKindMap[16]:
			if tot.Stats.LastNurse == nil {
				tot.Stats.LastNurse = t.Time
				tot.Stats.LastNurseSide = "L"
			}
		case totConfig.TallyKindMap[17]:
			if tot.Stats.LastNurse == nil {
				tot.Stats.LastNurse = t.Time
				tot.Stats.LastNurseSide = "R"
			}
		}
	}
}
