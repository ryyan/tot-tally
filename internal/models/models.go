// models.go defines domain models and template data structures.
package models

import "time"

// Tot is the core model representing a child's record.
type Tot struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Timezone       string         `json:"timezone"`
	MilkSetting    string         `json:"milkSetting"`
	Tallies        []Tally        `json:"tallies"`
	Stats          Stats          `json:"stats"`
	GeneratedStats GeneratedStats `json:"generatedStats"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// Tally represents a single recorded event.
type Tally struct {
	Time *time.Time `json:"time"`
	Kind string     `json:"kind"`
}

// Stats tracks the last time specific activities occurred.
type Stats struct {
	LastMilk      *time.Time `json:"lastMilk"`
	LastNurse     *time.Time `json:"lastNurse"`
	LastNurseSide string     `json:"lastNurseSide"`
	LastSnack     *time.Time `json:"lastSnack"`
	LastMeal      *time.Time `json:"lastMeal"`
	LastPee       *time.Time `json:"lastPee"`
	LastPoo       *time.Time `json:"lastPoo"`
	LastBath      *time.Time `json:"lastBath"`
	LastBrush     *time.Time `json:"lastBrush"`
}

// GeneratedStats holds pre-calculated totals/trends for the UI.
type GeneratedStats struct {
	Last12HoursMilk   string `json:"last12HoursMilk"`
	Last12HoursNurse  string `json:"last12HoursNurse"`
	Last12HoursPee    string `json:"last12HoursPee"`
	Last12HoursPoo    string `json:"last12HoursPoo"`
	Last24HoursMilk   string `json:"last24HoursMilk"`
	Last24HoursNurse  string `json:"last24HoursNurse"`
	Last24HoursPee    string `json:"last24HoursPee"`
	Last24HoursPoo    string `json:"last24HoursPoo"`
	TodayMilk         string `json:"todayMilk"`
	TodayNurse        string `json:"todayNurse"`
	TodayPee          string `json:"todayPee"`
	TodayPoo          string `json:"todayPoo"`
	YesterdayMilk     string `json:"yesterdayMilk"`
	YesterdayNurse    string `json:"yesterdayNurse"`
	YesterdayPee      string `json:"yesterdayPee"`
	YesterdayPoo      string `json:"yesterdayPoo"`
	TwoDaysAgoMilk    string `json:"twoDaysAgoMilk"`
	TwoDaysAgoNurse   string `json:"twoDaysAgoNurse"`
	TwoDaysAgoPee     string `json:"twoDaysAgoPee"`
	TwoDaysAgoPoo     string `json:"twoDaysAgoPoo"`
	ThreeDaysAgoMilk  string `json:"threeDaysAgoMilk"`
	ThreeDaysAgoNurse string `json:"threeDaysAgoNurse"`
	ThreeDaysAgoPee   string `json:"threeDaysAgoPee"`
	ThreeDaysAgoPoo   string `json:"threeDaysAgoPoo"`
	ThreeDayAvgMilk   string `json:"threeDayAvgMilk"`
	ThreeDayAvgNurse  string `json:"threeDayAvgNurse"`
	ThreeDayAvgPee    string `json:"threeDayAvgPee"`
	ThreeDayAvgPoo    string `json:"threeDayAvgPoo"`
	AvgGapMilk        string `json:"avgGapMilk"`
	AvgGapNurse       string `json:"avgGapNurse"`
	AvgGapPee         string `json:"avgGapPee"`
	AvgGapPoo         string `json:"avgGapPoo"`
}

// HomePageData is passed to the index.html template.
type HomePageData struct {
	FlashMessage string
	IsErrorFlash bool
}

// TotPageData is passed to the tot.html dashboard template.
type TotPageData struct {
	ID                 string
	Name               string
	Timezone           string
	MilkSetting        string
	MilkSettingDisplay string
	FlashMessage       string
	IsErrorFlash       bool
	Tallies            []TotPageTally
	Stats              TotPageStats
	GeneratedStats     GeneratedStats
	MaxTallies         int
}

type TotPageTally struct {
	Time string
	Kind string
}

type TotPageStats struct {
	LastMilk       string
	LastMilkAmount string
	LastNurse      string
	LastNurseSide  string
	LastSnack      string
	LastMeal       string
	LastPee        string
	LastPoo        string
	LastBath       string
	LastBrush      string
}
