package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// FileConcurrencyCache provides concurrency slot management using the filesystem
// as a fallback when Redis is unavailable. Uses flock for atomic locking.
type FileConcurrencyCache struct {
	lockDir string
	ttlSecs int

	// Per-process in-memory slot tracking to avoid excessive flock calls.
	// Key: "account:{id}" or "user:{id}", Value: set of requestIDs (simplified: just count)
	mu           sync.Mutex
	slotCounts   map[string]int                 // key -> current count
	slotOwners   map[string]map[string]struct{} // key -> set of requestIDs
	slotLastSeen map[string]time.Time           // key -> last access time for cleanup
}

const fileConcurrencyLockDir = "/var/run/sub2api/concurrency"

// NewFileConcurrencyCache creates a file-based concurrency cache.
// Falls back to /tmp if /var/run/sub2api doesn't exist.
func NewFileConcurrencyCache(slotTTLMinutes int) (*FileConcurrencyCache, error) {
	ttlSecs := slotTTLMinutes * 60
	if ttlSecs <= 0 {
		ttlSecs = 15 * 60
	}

	// Try /var/run/sub2api first, fall back to /tmp
	lockDir := fileConcurrencyLockDir
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		lockDir = filepath.Join(os.TempDir(), "sub2api-concurrency")
		if err := os.MkdirAll(lockDir, 0755); err != nil {
			return nil, fmt.Errorf("cannot create concurrency lock dir: %w", err)
		}
	}

	return &FileConcurrencyCache{
		lockDir:      lockDir,
		ttlSecs:      ttlSecs,
		slotCounts:   make(map[string]int),
		slotOwners:   make(map[string]map[string]struct{}),
		slotLastSeen: make(map[string]time.Time),
	}, nil
}

type slotFileData struct {
	Count int              `json:"count"`
	Slots map[string]int64 `json:"slots"` // requestID -> expiry unix timestamp
}

func (f *FileConcurrencyCache) slotFilePath(key string) string {
	return filepath.Join(f.lockDir, key+".slot")
}

func (f *FileConcurrencyCache) lockFilePath(key string) string {
	return filepath.Join(f.lockDir, key+".lock")
}

// AcquireAccountSlot acquires a concurrency slot for an account.
// Returns (acquired=true) if slot available, (acquired=false) if at limit.
func (f *FileConcurrencyCache) AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error) {
	return f.acquireSlot("account", accountID, maxConcurrency, requestID)
}

// AcquireUserSlot acquires a concurrency slot for a user.
func (f *FileConcurrencyCache) AcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int, requestID string) (bool, error) {
	return f.acquireSlot("user", userID, maxConcurrency, requestID)
}

func (f *FileConcurrencyCache) acquireSlot(kind string, id int64, maxConcurrency int, requestID string) (bool, error) {
	key := fmt.Sprintf("concurrency:%s:%d", kind, id)
	lockPath := f.lockFilePath(key)
	dataPath := f.slotFilePath(key)

	now := time.Now()

	// Try in-memory first (fast path for same-process concurrent calls)
	f.mu.Lock()
	if owners, ok := f.slotOwners[key]; ok {
		if _, present := owners[requestID]; present {
			// Already own this slot — refresh
			f.slotLastSeen[key] = now
			f.mu.Unlock()
			return true, nil
		}
		if len(owners) < maxConcurrency {
			owners[requestID] = struct{}{}
			f.slotCounts[key]++
			f.slotLastSeen[key] = now
			f.mu.Unlock()
			return true, nil
		}
		// At limit in memory
		f.mu.Unlock()
		// But we need to check disk to be accurate
	} else {
		f.mu.Unlock()
	}

	// Acquire exclusive lock on the lock file
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return false, fmt.Errorf("open lock file: %w", err)
	}
	defer func() { _ = lockFile.Close() }()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return false, fmt.Errorf("flock: %w", err)
	}
	defer func() { _ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN) }()

	// Read existing data
	var data slotFileData
	if raw, err := os.ReadFile(dataPath); err == nil {
		if err := json.Unmarshal(raw, &data); err != nil {
			data = slotFileData{}
		}
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("read slot data: %w", err)
	}
	if data.Slots == nil {
		data.Slots = make(map[string]int64)
	}

	// Clean expired slots
	for rid, expiry := range data.Slots {
		if expiry < now.Unix() {
			delete(data.Slots, rid)
		}
	}

	// Check if requestID already present
	if _, present := data.Slots[requestID]; present {
		// Refresh — update expiry
		data.Slots[requestID] = now.Add(time.Duration(f.ttlSecs) * time.Second).Unix()
		data.Count = len(data.Slots)
		if err := f.writeSlotData(dataPath, &data); err != nil {
			return false, err
		}
		f.updateMemory(key, requestID, now)
		return true, nil
	}

	// Check limit
	if len(data.Slots) >= maxConcurrency {
		return false, nil
	}

	// Acquire slot
	data.Slots[requestID] = now.Add(time.Duration(f.ttlSecs) * time.Second).Unix()
	data.Count = len(data.Slots)
	if err := f.writeSlotData(dataPath, &data); err != nil {
		return false, err
	}
	f.updateMemory(key, requestID, now)
	return true, nil
}

// ReleaseAccountSlot releases an account concurrency slot.
func (f *FileConcurrencyCache) ReleaseAccountSlot(ctx context.Context, accountID int64, requestID string) error {
	return f.releaseSlot("account", accountID, requestID)
}

// ReleaseUserSlot releases a user concurrency slot.
func (f *FileConcurrencyCache) ReleaseUserSlot(ctx context.Context, userID int64, requestID string) error {
	return f.releaseSlot("user", userID, requestID)
}

func (f *FileConcurrencyCache) releaseSlot(kind string, id int64, requestID string) error {
	key := fmt.Sprintf("concurrency:%s:%d", kind, id)
	lockPath := f.lockFilePath(key)
	dataPath := f.slotFilePath(key)

	// Remove from memory first (best effort)
	f.mu.Lock()
	if owners, ok := f.slotOwners[key]; ok {
		delete(owners, requestID)
		f.slotCounts[key]--
		if len(owners) == 0 {
			delete(f.slotOwners, key)
			delete(f.slotCounts, key)
			delete(f.slotLastSeen, key)
		}
	}
	f.mu.Unlock()

	// Acquire lock and remove from file
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	defer func() { _ = lockFile.Close() }()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock: %w", err)
	}
	defer func() { _ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN) }()

	var data slotFileData
	if raw, err := os.ReadFile(dataPath); err == nil {
		if err := json.Unmarshal(raw, &data); err != nil {
			data = slotFileData{}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read slot data: %w", err)
	}

	delete(data.Slots, requestID)
	data.Count = len(data.Slots)

	if data.Count == 0 {
		if err := os.Remove(dataPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove slot data: %w", err)
		}
	} else {
		if err := f.writeSlotData(dataPath, &data); err != nil {
			return err
		}
	}
	return nil
}

func (f *FileConcurrencyCache) writeSlotData(path string, data *slotFileData) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal slot data: %w", err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		return fmt.Errorf("write slot data: %w", err)
	}
	return nil
}

func (f *FileConcurrencyCache) updateMemory(key, requestID string, now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.slotOwners[key] == nil {
		f.slotOwners[key] = make(map[string]struct{})
	}
	if _, exists := f.slotOwners[key][requestID]; !exists {
		f.slotOwners[key][requestID] = struct{}{}
		f.slotCounts[key]++
	}
	f.slotLastSeen[key] = now
}

// CleanupExpiredSlots removes stale slot files older than ttl.
// Called periodically or on startup.
func (f *FileConcurrencyCache) CleanupExpiredSlots() error {
	entries, err := os.ReadDir(f.lockDir)
	if err != nil {
		return fmt.Errorf("read lock dir: %w", err)
	}

	now := time.Now()
	cutoff := now.Add(-time.Duration(f.ttlSecs) * time.Second)

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".slot" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		// If file hasn't been modified in ttl*2, consider it stale
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(f.lockDir, entry.Name()))
			// Also remove corresponding lock file
			lockName := entry.Name()[:len(entry.Name())-5] + ".lock"
			_ = os.Remove(filepath.Join(f.lockDir, lockName))
		}
	}
	return nil
}

// Stats returns current lock directory stats for monitoring.
func (f *FileConcurrencyCache) Stats() (totalSlots int, lockFiles int) {
	entries, err := os.ReadDir(f.lockDir)
	if err != nil {
		return 0, 0
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".lock" {
			lockFiles++
		}
	}
	f.mu.Lock()
	for _, count := range f.slotCounts {
		totalSlots += count
	}
	f.mu.Unlock()
	return totalSlots, lockFiles
}

// LockDir returns the directory being used for file locks.
func (f *FileConcurrencyCache) LockDir() string {
	return f.lockDir
}
