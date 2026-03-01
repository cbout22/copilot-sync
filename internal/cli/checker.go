package cli

import (
	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/injector"
	"github.com/cbout22/copilot-sync/internal/manifest"
)

// CheckStatus describes the sync status of a single asset.
type CheckStatus int

const (
	CheckOK          CheckStatus = iota // File exists, lock matches
	CheckNeverSynced                    // Not in lock, not on disk
	CheckFileMissing                    // In lock but file deleted
	CheckNotInLock                      // File exists but no lock entry
	CheckRefMismatch                    // Lock ref differs from manifest ref
)

// CheckResult holds the outcome of checking one asset entry.
type CheckResult struct {
	Type     string
	Name     string
	Status   CheckStatus
	LockRef  string // ref in lock file (empty if not in lock)
	ManifRef string // ref in manifest
}

// CheckAssets validates all entries against the lock file and filesystem.
// This is a pure function: it reads state through its arguments, not globals.
func CheckAssets(entries []manifest.Entry, lock *manifest.LockFile, fs injector.FileWriter) []CheckResult {
	results := make([]CheckResult, 0, len(entries))

	for _, entry := range entries {
		assetType := config.AssetType(entry.Type)
		targetPath := assetType.TargetPath(entry.Name)

		fileExists := fs.Exists(targetPath)
		lockEntry, locked := lock.Get(entry.Type, entry.Name)

		var status CheckStatus
		switch {
		case !fileExists && !locked:
			status = CheckNeverSynced
		case !fileExists && locked:
			status = CheckFileMissing
		case fileExists && !locked:
			status = CheckNotInLock
		case fileExists && locked && lockEntry.Ref != entry.Ref:
			status = CheckRefMismatch
		default:
			status = CheckOK
		}

		results = append(results, CheckResult{
			Type:     entry.Type,
			Name:     entry.Name,
			Status:   status,
			LockRef:  lockEntry.Ref,
			ManifRef: entry.Ref,
		})
	}

	return results
}
