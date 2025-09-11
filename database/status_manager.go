package database

import (
	"discord-bot/models"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StatusManager manages the database status file.
type StatusManager struct {
	statusFile string
	mutex      sync.Mutex
	status     *models.DBStatus
}

// NewStatusManager creates a new status manager.
func NewStatusManager(statusFile string) *StatusManager {
	return &StatusManager{
		statusFile: statusFile,
		status: &models.DBStatus{
			ActiveDatabases: make(map[string]*models.GuildDatabases),
		},
	}
}

// RegisterDB registers a database instance for a specific guild and mode.
func (sm *StatusManager) RegisterDB(guildID, mode, dbPath string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, ok := sm.status.ActiveDatabases[guildID]; !ok {
		sm.status.ActiveDatabases[guildID] = &models.GuildDatabases{}
	}

	dbInfo := &models.DatabaseInfo{
		DBPath: dbPath,
		Status: "active",
	}

	switch mode {
	case "base":
		sm.status.ActiveDatabases[guildID].Base = dbInfo
	case "plus":
		sm.status.ActiveDatabases[guildID].Plus = dbInfo
	}
}

// Save commits the current database status to the JSON file.
func (sm *StatusManager) Save() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.status.LastUpdated = time.Now()

	// Ensure the directory exists.
	dir := filepath.Dir(sm.statusFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create status directory: %w", err)
	}

	// Marshal the data to JSON.
	data, err := json.MarshalIndent(sm.status, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	// Write the file, overwriting it if it exists.
	if err := os.WriteFile(sm.statusFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write status file: %w", err)
	}

	return nil
}
