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

// StatusManager manages the database status file
type StatusManager struct {
	statusFile string
	mutex      sync.RWMutex
}

// NewStatusManager creates a new status manager
func NewStatusManager(statusFile string) *StatusManager {
	return &StatusManager{
		statusFile: statusFile,
	}
}

// LoadStatus loads the database status from file
func (sm *StatusManager) LoadStatus() (*models.DatabaseStatus, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Check if file exists
	if _, err := os.Stat(sm.statusFile); os.IsNotExist(err) {
		// Return empty status if file doesn't exist
		return &models.DatabaseStatus{
			CurrentDatabases: []models.DatabaseInfo{},
			TotalDatabases:   0,
			LastUpdated:      time.Now(),
		}, nil
	}

	data, err := os.ReadFile(sm.statusFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read status file: %w", err)
	}

	var status models.DatabaseStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status file: %w", err)
	}

	return &status, nil
}

// SaveStatus saves the database status to file
func (sm *StatusManager) SaveStatus(status *models.DatabaseStatus) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(sm.statusFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create status directory: %w", err)
	}

	status.LastUpdated = time.Now()

	data, err := json.MarshalIndent(status, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	if err := os.WriteFile(sm.statusFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write status file: %w", err)
	}

	return nil
}

// UpdateDatabaseInfo updates or adds database information
func (sm *StatusManager) UpdateDatabaseInfo(guildID, dbFile string, messageCount int64) error {
	status, err := sm.LoadStatus()
	if err != nil {
		return err
	}

	// Find existing database info
	found := false
	for i, dbInfo := range status.CurrentDatabases {
		if dbInfo.GuildID == guildID && dbInfo.DBFile == dbFile {
			// Update existing entry
			status.CurrentDatabases[i].MessageCount = messageCount
			status.CurrentDatabases[i].LastMessage = time.Now()
			found = true
			break
		}
	}

	// Add new entry if not found
	if !found {
		newDB := models.DatabaseInfo{
			GuildID:      guildID,
			DBFile:       dbFile,
			CreatedAt:    time.Now(),
			MessageCount: messageCount,
			LastMessage:  time.Now(),
		}
		status.CurrentDatabases = append(status.CurrentDatabases, newDB)
		status.TotalDatabases++
	}

	return sm.SaveStatus(status)
}

// RemoveDatabaseInfo removes database information
func (sm *StatusManager) RemoveDatabaseInfo(guildID, dbFile string) error {
	status, err := sm.LoadStatus()
	if err != nil {
		return err
	}

	// Find and remove the database info
	for i, dbInfo := range status.CurrentDatabases {
		if dbInfo.GuildID == guildID && dbInfo.DBFile == dbFile {
			// Remove the element
			status.CurrentDatabases = append(status.CurrentDatabases[:i], status.CurrentDatabases[i+1:]...)
			if status.TotalDatabases > 0 {
				status.TotalDatabases--
			}
			break
		}
	}

	return sm.SaveStatus(status)
}

// GetDatabaseInfo retrieves database information for a specific guild
func (sm *StatusManager) GetDatabaseInfo(guildID string) ([]models.DatabaseInfo, error) {
	status, err := sm.LoadStatus()
	if err != nil {
		return nil, err
	}

	var result []models.DatabaseInfo
	for _, dbInfo := range status.CurrentDatabases {
		if dbInfo.GuildID == guildID {
			result = append(result, dbInfo)
		}
	}

	return result, nil
}

// GetAllDatabaseInfo retrieves all database information
func (sm *StatusManager) GetAllDatabaseInfo() ([]models.DatabaseInfo, error) {
	status, err := sm.LoadStatus()
	if err != nil {
		return nil, err
	}

	return status.CurrentDatabases, nil
}

// CleanupOldDatabases removes database entries that haven't been updated for a specified duration
func (sm *StatusManager) CleanupOldDatabases(maxAge time.Duration) error {
	status, err := sm.LoadStatus()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	var activeDatabases []models.DatabaseInfo

	for _, dbInfo := range status.CurrentDatabases {
		if dbInfo.LastMessage.After(cutoff) {
			activeDatabases = append(activeDatabases, dbInfo)
		}
	}

	status.CurrentDatabases = activeDatabases
	status.TotalDatabases = len(activeDatabases)

	return sm.SaveStatus(status)
}