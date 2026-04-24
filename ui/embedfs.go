package ui

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Dist contains the contents of the built UI under dist/.
//
//go:embed dist/*
var Dist embed.FS

var assets http.FileSystem

func init() {
	sub, err := fs.Sub(Dist, "dist")
	if err != nil {
		// If the build is misconfigured and dist/ is missing, fail fast.
		panic(err)
	}
	assets = http.FS(sub)
}

// EnforceFreshBuild exits the process with status 1 if the embedded UI build
// is stale relative to ui/src on disk.
//
// Call this only from commands that actually serve the UI (i.e. `serve`) —
// not from CLI subcommands, because that would break scheduled invocations
// (e.g. `shelley client chat` from a systemd timer) whenever a developer
// edits ui/src without rebuilding.
func EnforceFreshBuild() {
	buildInfoData, err := fs.ReadFile(Dist, "dist/build-info.json")
	if err != nil {
		staleExit("dist/build-info.json is missing from the embedded bundle", "", nil)
	}
	var buildInfo struct {
		Timestamp int64  `json:"timestamp"`
		Date      string `json:"date"`
		SrcDir    string `json:"srcDir"`
	}
	if err := json.Unmarshal(buildInfoData, &buildInfo); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse build-info.json: %v\n", err)
		return
	}
	srcDir := buildInfo.SrcDir
	if srcDir == "" {
		return
	}
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		// Deployed binary: source tree not on this machine.
		return
	}
	buildTime := time.UnixMilli(buildInfo.Timestamp)
	var newerFiles []string
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.ModTime().After(buildTime) {
			newerFiles = append(newerFiles, path)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to check source file timestamps: %v\n", err)
		return
	}
	if len(newerFiles) > 0 {
		staleExit("", buildInfo.Date, newerFiles)
	}
}

func staleExit(reason, buildDate string, newerFiles []string) {
	fmt.Fprintf(os.Stderr, "\nError: UI build is stale!\n")
	if buildDate != "" {
		fmt.Fprintf(os.Stderr, "Build timestamp: %s\n", buildDate)
	}
	if reason != "" {
		fmt.Fprintf(os.Stderr, "Reason: %s\n", reason)
	}
	if len(newerFiles) > 0 {
		fmt.Fprintf(os.Stderr, "\nThe following source files are newer than the build:\n")
		for _, f := range newerFiles {
			fmt.Fprintf(os.Stderr, "  - %s\n", f)
		}
	}
	fmt.Fprintf(os.Stderr, "\nRebuild the UI first: cd ui && pnpm run build\n\n")
	os.Exit(1)
}

// Assets returns an http.FileSystem backed by the embedded UI assets.
func Assets() http.FileSystem {
	return assets
}

// Checksums returns the content checksums for static assets.
// These are computed during build and used for ETag generation.
func Checksums() map[string]string {
	data, err := fs.ReadFile(Dist, "dist/checksums.json")
	if err != nil {
		return nil
	}
	var checksums map[string]string
	if err := json.Unmarshal(data, &checksums); err != nil {
		return nil
	}
	return checksums
}
