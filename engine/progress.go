package engine

import "github.com/Anastylosis/MSD/site"

// ProgressReporter receives download progress events from an Engine.
type ProgressReporter interface {
	OnFileStart(file site.File)
	OnFileProgress(file site.File, bytesDownloaded, totalBytes int64)
	OnFileComplete(file site.File, err error)
	OnAlbumComplete(album site.Album, succeeded, failed int)
}

// NoopReporter is a ProgressReporter that discards all events.
type NoopReporter struct{}

// OnFileStart implements ProgressReporter.
func (NoopReporter) OnFileStart(site.File) {}

// OnFileProgress implements ProgressReporter.
func (NoopReporter) OnFileProgress(site.File, int64, int64) {}

// OnFileComplete implements ProgressReporter.
func (NoopReporter) OnFileComplete(site.File, error) {}

// OnAlbumComplete implements ProgressReporter.
func (NoopReporter) OnAlbumComplete(site.Album, int, int) {}
