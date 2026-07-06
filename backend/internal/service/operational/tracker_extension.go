package operational

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var ErrTrackerExtensionUnavailable = errors.New("tracker extension package is unavailable")

// ExtensionStatus tells a client whether its installed extension is current.
type ExtensionStatus struct {
	LatestVersion   string `json:"latest_version"`
	ReportedVersion string `json:"reported_version"`
	NeedsUpdate     bool   `json:"needs_update"`
}

type extensionManifest struct {
	Version string `json:"version"`
}

// LatestExtensionVersion reads the version from the bundled extension manifest —
// i.e. the build the server hands out on download. Single source of truth: bump
// the manifest and the "latest" everyone compares against moves with it.
func (s *TrackerService) LatestExtensionVersion() (string, error) {
	dir, err := resolveExtensionDir()
	if err != nil {
		return "", ErrTrackerExtensionUnavailable
	}
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return "", ErrTrackerExtensionUnavailable
	}
	var manifest extensionManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", ErrTrackerExtensionUnavailable
	}
	return strings.TrimSpace(manifest.Version), nil
}

// GetExtensionStatus compares the version last reported by the user's extension
// against the bundled build so the dashboard can prompt a re-download.
func (s *TrackerService) GetExtensionStatus(ctx context.Context, userID string) (ExtensionStatus, error) {
	latest, err := s.LatestExtensionVersion()
	if err != nil {
		return ExtensionStatus{}, err
	}
	reported, err := s.repo.GetUserExtensionVersion(ctx, userID)
	if err != nil {
		return ExtensionStatus{}, err
	}
	return ExtensionStatus{
		LatestVersion:   latest,
		ReportedVersion: reported,
		NeedsUpdate:     reported != "" && isExtensionVersionOlder(reported, latest),
	}, nil
}

// isExtensionVersionOlder compares dotted numeric versions ("1.9" < "1.10").
func isExtensionVersionOlder(current, latest string) bool {
	a := parseVersionParts(current)
	b := parseVersionParts(latest)
	length := len(a)
	if len(b) > length {
		length = len(b)
	}
	for i := 0; i < length; i++ {
		var left, right int
		if i < len(a) {
			left = a[i]
		}
		if i < len(b) {
			right = b[i]
		}
		if left < right {
			return true
		}
		if left > right {
			return false
		}
	}
	return false
}

func parseVersionParts(version string) []int {
	fields := strings.Split(strings.TrimSpace(version), ".")
	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		value, _ := strconv.Atoi(strings.TrimSpace(field))
		parts = append(parts, value)
	}
	return parts
}

// BuildExtensionArchive zips the bundled extension. The base manifest targets
// Chromium (background.service_worker); for browser=="firefox" the manifest is
// rewritten to a background.scripts event page, since Firefox does not support
// service workers and Chrome rejects background.scripts in MV3.
func (s *TrackerService) BuildExtensionArchive(ctx context.Context, browser string) ([]byte, string, error) {
	extensionDir, err := resolveExtensionDir()
	if err != nil {
		slog.ErrorContext(ctx, "failed to resolve extension directory", "error", err)
		return nil, "", ErrTrackerExtensionUnavailable
	}
	slog.InfoContext(ctx, "resolved extension directory", "path", extensionDir)

	buffer := bytes.NewBuffer(nil)
	archive := zip.NewWriter(buffer)

	if err := filepath.WalkDir(extensionDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(extensionDir, path)
		if err != nil {
			return fmt.Errorf("resolve relative extension path: %w", err)
		}

		zipPath := filepath.ToSlash(relativePath)
		writer, err := archive.Create(zipPath)
		if err != nil {
			return fmt.Errorf("create zip entry %s: %w", zipPath, err)
		}

		// Rewrite the manifest for Firefox instead of copying it verbatim.
		if zipPath == "manifest.json" && strings.EqualFold(browser, "firefox") {
			raw, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read manifest for firefox: %w", err)
			}
			transformed, err := firefoxManifest(raw)
			if err != nil {
				return fmt.Errorf("rewrite manifest for firefox: %w", err)
			}
			if _, err := writer.Write(transformed); err != nil {
				return fmt.Errorf("write firefox manifest: %w", err)
			}
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open extension file %s: %w", path, err)
		}
		defer file.Close()

		if _, err := io.Copy(writer, file); err != nil {
			return fmt.Errorf("write extension file %s: %w", path, err)
		}

		return nil
	}); err != nil {
		_ = archive.Close()
		slog.ErrorContext(ctx, "failed to walk extension directory", "error", err, "dir", extensionDir)
		return nil, "", fmt.Errorf("archive tracker extension: %w", err)
	}

	if err := archive.Close(); err != nil {
		return nil, "", fmt.Errorf("finalize tracker extension archive: %w", err)
	}

	filename := "kantor-activity-tracker.zip"
	if strings.EqualFold(browser, "firefox") {
		filename = "kantor-activity-tracker-firefox.zip"
	}
	return buffer.Bytes(), filename, nil
}

// firefoxManifest rewrites the Chromium manifest's background key to a Firefox
// event page (background.scripts), leaving everything else untouched.
func firefoxManifest(raw []byte) ([]byte, error) {
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	manifest["background"] = map[string]any{"scripts": []string{"background.js"}}
	return json.MarshalIndent(manifest, "", "  ")
}

func resolveExtensionDir() (string, error) {
	candidates := []string{
		"extension",
		filepath.Join("..", "extension"),
		filepath.Join("..", "..", "extension"),
		filepath.Join("..", "..", "..", "extension"),
		filepath.Join(string(filepath.Separator), "app", "extension"),
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue
		}

		manifestPath := filepath.Join(candidate, "manifest.json")
		if _, err := os.Stat(manifestPath); err != nil {
			continue
		}

		absolutePath, err := filepath.Abs(candidate)
		if err != nil {
			return "", fmt.Errorf("resolve extension path: %w", err)
		}

		return absolutePath, nil
	}

	return "", errors.New("extension directory not found")
}

func sanitizeContentDispositionFilename(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "download.zip"
	}

	return strings.ReplaceAll(trimmed, "\"", "")
}
