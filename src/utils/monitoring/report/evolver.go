package report

import (
	"go.uber.org/atomic"
)

type EvolverErrors struct {
	StoreDbError            atomic.Uint64 `json:"store_sources_saved_error"`
	PollerFetchError        atomic.Uint64 `json:"poller_fetch_error"`
	DownloaderDownlaodError atomic.Uint64 `json:"downloader_download_error"`
}

type EvolverState struct {
	// Counting missing evolved sources
	PollerSourcesFromSelects atomic.Uint64 `json:"poller_sources_from_selects"`

	// Counting evolved sources saved to db
	StoreSourcesSaved atomic.Uint64 `json:"store_sources_saved"`

	// Counting downloaded evolved sources
	DownloaderSourcesLoaded atomic.Uint64 `json:"downloader_sources_loaded"`
}

type EvolverReport struct {
	State  EvolverState  `json:"state"`
	Errors EvolverErrors `json:"errors"`
}
