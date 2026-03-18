// core.go is the business logic coordinator for tot management. It ties storage, stats, and web together.
package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	totConfig "tot-tally/internal/config"
	totModels "tot-tally/internal/models"
	totStats "tot-tally/internal/stats"
	totStorage "tot-tally/internal/storage"
)

// Service coordinates high-level business operations.
type Service struct {
	config *totConfig.Config
	store  *totStorage.Repository
	stats  *totStats.Engine
}

// NewService initializes the business logic layer with its requirements.
func NewService(cfg *totConfig.Config, store *totStorage.Repository, engine *totStats.Engine) *Service {
	return &Service{config: cfg, store: store, stats: engine}
}

// CreateTot initializes and persists a new child record.
func (s *Service) CreateTot(name, timezone, milkSetting string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 15 {
		return "", errors.New("invalid name length")
	}

	tzLocation, err := time.LoadLocation(timezone)
	if err != nil {
		return "", fmt.Errorf("core: invalid timezone %q: %w", timezone, err)
	}

	var newID string
	for {
		newID, err = s.store.GenerateID()
		if err != nil {
			return "", fmt.Errorf("core: id generation failed: %w", err)
		}

		path := filepath.Join(s.config.TotDirectory, newID+".json")
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("core: checking id availability: %w", err)
		}
	}

	now := time.Now().UTC()
	newTot := totModels.Tot{
		ID:          newID,
		Name:        name,
		Timezone:    timezone,
		MilkSetting: milkSetting,
		Tallies:     []totModels.Tally{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	generated, err := s.stats.GenerateStats(&newTot, tzLocation, time.Now())
	if err != nil {
		return "", fmt.Errorf("core: initial stats failed: %w", err)
	}
	newTot.GeneratedStats = generated

	if err := s.store.SaveTot(&newTot); err != nil {
		return "", fmt.Errorf("core: persistence failed: %w", err)
	}
	return newID, nil
}

// AddTally records a new activity event.
func (s *Service) AddTally(tot *totModels.Tot, kindKey string) error {
	kindKeyInt, err := strconv.ParseInt(kindKey, 10, 64)
	if err != nil {
		return fmt.Errorf("core: key format: %w", err)
	}

	kind, exists := totConfig.TallyKindMap[kindKeyInt]
	if !exists {
		return fmt.Errorf("core: unknown kind: %d", kindKeyInt)
	}

	now := time.Now().UTC()
	if kind == totConfig.TallyKindMap[13] {
		pee := totModels.Tally{Time: &now, Kind: totConfig.TallyKindMap[11]}
		poo := totModels.Tally{Time: &now, Kind: totConfig.TallyKindMap[12]}
		tot.Tallies = append([]totModels.Tally{pee, poo}, tot.Tallies...)
	} else {
		tot.Tallies = append([]totModels.Tally{{Time: &now, Kind: kind}}, tot.Tallies...)
	}

	s.updateLatestMarkers(tot, kind, &now)
	return nil
}

func (s *Service) updateLatestMarkers(tot *totModels.Tot, kind string, now *time.Time) {
	if strings.HasPrefix(kind, "🍼") {
		tot.Stats.LastMilk = now
		return
	}
	switch kind {
	case totConfig.TallyKindMap[9]:
		tot.Stats.LastSnack = now
	case totConfig.TallyKindMap[10]:
		tot.Stats.LastMeal = now
	case totConfig.TallyKindMap[11]:
		tot.Stats.LastPee = now
	case totConfig.TallyKindMap[12]:
		tot.Stats.LastPoo = now
	case totConfig.TallyKindMap[13]:
		tot.Stats.LastPee, tot.Stats.LastPoo = now, now
	case totConfig.TallyKindMap[14]:
		tot.Stats.LastBath = now
	case totConfig.TallyKindMap[15]:
		tot.Stats.LastBrush = now
	case totConfig.TallyKindMap[16]:
		tot.Stats.LastNurse = now
		tot.Stats.LastNurseSide = "L"
	case totConfig.TallyKindMap[17]:
		tot.Stats.LastNurse = now
		tot.Stats.LastNurseSide = "R"
	}
}
