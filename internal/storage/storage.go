// storage.go handles atomic file-based persistence and IP-based rate limiting.
package storage

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	totConfig "tot-tally/internal/config"
	totModels "tot-tally/internal/models"
	totShards "tot-tally/internal/shards"

	"github.com/google/uuid"
)

// Repository handles the atomic file persistence for the application.
type Repository struct {
	config *totConfig.Config
	pool   *totShards.Pool
}

// NewRepository initializes a data store with its dependencies.
func NewRepository(cfg *totConfig.Config, pool *totShards.Pool) *Repository {
	return &Repository{config: cfg, pool: pool}
}

// SaveTot writes the record to disk atomically using Write-Then-Rename.
func (r *Repository) SaveTot(tot *totModels.Tot) error {
	if len(tot.Tallies) > r.config.MaxTallies {
		tot.Tallies = tot.Tallies[:r.config.MaxTallies]
	}
	tot.UpdatedAt = time.Now().UTC()

	finalPath := filepath.Join(r.config.TotDirectory, filepath.Base(tot.ID)+".json")
	tmpPath := finalPath + ".tmp"

	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("storage: failed to create tmp file: %w", err)
	}

	if err := json.NewEncoder(file).Encode(tot); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("storage: failed to encode tot: %w", err)
	}
	file.Close()

	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("storage: failed to swap file: %w", err)
	}
	return nil
}

// LoadTot reads a child record from disk.
func (r *Repository) LoadTot(totID string) (*totModels.Tot, error) {
	filename := filepath.Join(r.config.TotDirectory, filepath.Base(totID)+".json")
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("tot does not exist")
		}
		return nil, fmt.Errorf("storage: failed to open file: %w", err)
	}
	defer file.Close()

	tot := &totModels.Tot{}
	if err := json.NewDecoder(file).Decode(tot); err != nil {
		return nil, fmt.Errorf("storage: failed to decode tot: %w", err)
	}

	if len(tot.Tallies) > r.config.MaxTallies {
		tot.Tallies = tot.Tallies[:r.config.MaxTallies]
	}
	if tot.MilkSetting == "" {
		tot.MilkSetting = "both"
	}
	return tot, nil
}

// CheckAndIncrementIPLimit manages the file-based IP counter.
func (r *Repository) CheckAndIncrementIPLimit(ip string) error {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(ip)))
	finalPath := filepath.Join(r.config.LimitDirectory, hash)
	tmpPath := finalPath + ".tmp"

	mut := r.pool.GetShardMutex(hash)
	mut.Lock()
	defer mut.Unlock()

	count := 0
	if data, err := os.ReadFile(finalPath); err == nil {
		lines := strings.Split(string(data), "\n")
		if len(lines) > 0 {
			count, _ = strconv.Atoi(lines[0])
		}
	}

	if count >= r.config.MaxTotsPerIP {
		return errors.New("limit reached")
	}

	content := fmt.Sprintf("%d\n%d", count+1, time.Now().UnixMilli())
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("storage: failed to write limit tmp: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("storage: failed to swap limit file: %w", err)
	}
	return nil
}

// GenerateID creates a new time-ordered UUID v7 string.
// UUID v7 is preferred because it is time-ordered and industry standard.
func (r *Repository) GenerateID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("storage: failed to generate uuid v7: %w", err)
	}
	return id.String(), nil
}
