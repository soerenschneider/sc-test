package transmission

import (
	"fmt"
	"time"
)

type TransmissionRequest struct {
	Method    string      `json:"method"`
	Arguments interface{} `json:"arguments,omitempty"`
	Tag       int         `json:"tag,omitempty"`
}

// TransmissionResponse represents the response from Transmission.
type answerPayload struct {
	Arguments interface{} `json:"arguments"`
	Result    string      `json:"result"`
	Tag       *int        `json:"tag"`
}

func (c *TransmissionClient) GetSessionId() string {
	defer c.sem.Unlock()
	c.sem.Lock()
	return c.sessionId
}

func (c *TransmissionClient) SetSessionId(sessionId string) {
	defer c.sem.Unlock()
	c.sem.Lock()
	c.sessionId = sessionId
}

type torrentGetResults struct {
	Torrents []Torrent `json:"torrents"`
}

type Torrents []Torrent

func (t *Torrents) AsNiceList() []string {
	ret := make([]string, len(*t))

	for idx, entry := range *t {
		ret[idx] = entry.String()
	}

	return ret
}

func (t *Torrents) AsTable() ([]string, [][]string) {
	headers := []string{
		"Id",
		"Name",
		"Status",
		"Percent",
	}

	ret := make([][]string, 0, len(*t))
	for _, torrent := range *t {
		ret = append(ret, []string{fmt.Sprintf("%d", *torrent.ID), *torrent.Name, torrent.Status.String(), fmt.Sprintf("%3.0f%%", *torrent.PercentDone*100)})
	}

	return headers, ret
}

func (t *Torrent) String() string {
	var name string
	if t.Name != nil {
		name = *t.Name
	}

	var id int64
	if t.ID != nil {
		id = *t.ID
	}

	var status = "Unknown"
	if t.Status != nil {
		status = t.Status.String()
	}

	return fmt.Sprintf("%d %s %s %3.0f%%", id, name, status, *t.PercentDone*100)
}

// Torrent represents all the possible fields of data for a torrent.
// All fields are pointers to detect if the value is nil (field not requested) or real default value.
type Torrent struct {
	ActivityDate            *time.Time        `json:"activityDate"`
	AddedDate               *time.Time        `json:"addedDate"`
	Availability            []int64           `json:"availability"` // RPC v17
	BandwidthPriority       *int64            `json:"bandwidthPriority"`
	Comment                 *string           `json:"comment"`
	CorruptEver             *int64            `json:"corruptEver"`
	Creator                 *string           `json:"creator"`
	DateCreated             *time.Time        `json:"dateCreated"`
	DesiredAvailable        *int64            `json:"desiredAvailable"`
	DoneDate                *time.Time        `json:"doneDate"`
	DownloadDir             *string           `json:"downloadDir"`
	DownloadedEver          *int64            `json:"downloadedEver"`
	DownloadLimit           *int64            `json:"downloadLimit"`
	DownloadLimited         *bool             `json:"downloadLimited"`
	EditDate                *time.Time        `json:"editDate"`
	Error                   *int64            `json:"error"`
	ErrorString             *string           `json:"errorString"`
	ETA                     *int64            `json:"eta"`
	ETAIdle                 *int64            `json:"etaIdle"`
	FileCount               *int64            `json:"file-count"` // RPC v17
	Files                   []TorrentFile     `json:"files"`
	FileStats               []TorrentFileStat `json:"fileStats"`
	Group                   *string           `json:"group"` // RPC v17
	HashString              *string           `json:"hashString"`
	HaveUnchecked           *int64            `json:"haveUnchecked"`
	HaveValid               *int64            `json:"haveValid"`
	HonorsSessionLimits     *bool             `json:"honorsSessionLimits"`
	ID                      *int64            `json:"id"`
	IsFinished              *bool             `json:"isFinished"`
	IsPrivate               *bool             `json:"isPrivate"`
	IsStalled               *bool             `json:"isStalled"`
	Labels                  []string          `json:"labels"` // RPC v16
	LeftUntilDone           *int64            `json:"leftUntilDone"`
	MagnetLink              *string           `json:"magnetLink"`
	ManualAnnounceTime      *int64            `json:"manualAnnounceTime"`
	MaxConnectedPeers       *int64            `json:"maxConnectedPeers"`
	MetadataPercentComplete *float64          `json:"metadataPercentComplete"`
	Name                    *string           `json:"name"`
	PeerLimit               *int64            `json:"peer-limit"`
	Peers                   []Peer            `json:"peers"`
	PeersConnected          *int64            `json:"peersConnected"`
	PeersFrom               *TorrentPeersFrom `json:"peersFrom"`
	PeersGettingFromUs      *int64            `json:"peersGettingFromUs"`
	PeersSendingToUs        *int64            `json:"peersSendingToUs"`
	PercentComplete         *float64          `json:"percentComplete"` // RPC v17
	PercentDone             *float64          `json:"percentDone"`
	Pieces                  *string           `json:"pieces"`
	PieceCount              *int64            `json:"pieceCount"`
	PieceSize               *uint64           `json:"PieceSize"`
	Priorities              []int64           `json:"priorities"`
	PrimaryMimeType         *string           `json:"primary-mime-type"` // RPC v17
	QueuePosition           *int64            `json:"queuePosition"`
	RateDownload            *int64            `json:"rateDownload"` // B/s
	RateUpload              *int64            `json:"rateUpload"`   // B/s
	RecheckProgress         *float64          `json:"recheckProgress"`
	TimeDownloading         *time.Duration    `json:"secondsDownloading"`
	TimeSeeding             *time.Duration    `json:"secondsSeeding"`
	SeedIdleLimit           *time.Duration    `json:"seedIdleLimit"`
	SeedIdleMode            *int64            `json:"seedIdleMode"`
	SeedRatioLimit          *float64          `json:"seedRatioLimit"`
	SeedRatioMode           *SeedRatioMode    `json:"seedRatioMode"`
	SizeWhenDone            *uint64           `json:"sizeWhenDone"`
	StartDate               *time.Time        `json:"startDate"`
	Status                  *TorrentStatus    `json:"status"`
	Trackers                []Tracker         `json:"trackers"`
	TrackerList             *string           `json:"trackerList"`
	TrackerStats            []TrackerStats    `json:"trackerStats"`
	TotalSize               *uint64           `json:"totalSize"`
	TorrentFile             *string           `json:"torrentFile"`
	UploadedEver            *int64            `json:"uploadedEver"`
	UploadLimit             *int64            `json:"uploadLimit"`
	UploadLimited           *bool             `json:"uploadLimited"`
	UploadRatio             *float64          `json:"uploadRatio"`
	Wanted                  []bool            `json:"wanted"`
	WebSeeds                []string          `json:"webseeds"`
	WebSeedsSendingToUs     *int64            `json:"webseedsSendingToUs"`
}

// TorrentPeersFrom represents the peers statistics of a torrent.
type TorrentPeersFrom struct {
	FromCache    int64 `json:"fromCache"`
	FromDHT      int64 `json:"fromDht"`
	FromIncoming int64 `json:"fromIncoming"`
	FromLPD      int64 `json:"fromLpd"`
	FromLTEP     int64 `json:"fromLtep"`
	FromPEX      int64 `json:"fromPex"`
	FromTracker  int64 `json:"fromTracker"`
}

type TorrentFile struct {
	BytesCompleted int64  `json:"bytesCompleted"`
	Length         int64  `json:"length"`
	Name           string `json:"name"`
}

// TorrentFileStat represents the metadata of a torrent's file.
type TorrentFileStat struct {
	BytesCompleted int64 `json:"bytesCompleted"`
	Wanted         bool  `json:"wanted"`
	Priority       int64 `json:"priority"`
}

// Peer represent a peer metadata of a torrent's peer list.
type Peer struct {
	Address            string  `json:"address"`
	ClientName         string  `json:"clientName"`
	ClientIsChoked     bool    `json:"clientIsChoked"`
	ClientIsInterested bool    `json:"clientIsInterested"`
	FlagStr            string  `json:"flagStr"`
	IsDownloadingFrom  bool    `json:"isDownloadingFrom"`
	IsEncrypted        bool    `json:"isEncrypted"`
	IsIncoming         bool    `json:"isIncoming"`
	IsUploadingTo      bool    `json:"isUploadingTo"`
	IsUTP              bool    `json:"isUTP"`
	PeerIsChoked       bool    `json:"peerIsChoked"`
	PeerIsInterested   bool    `json:"peerIsInterested"`
	Port               int64   `json:"port"`
	Progress           float64 `json:"progress"`
	RateToClient       int64   `json:"rateToClient"` // B/s
	RateToPeer         int64   `json:"rateToPeer"`   // B/s
}

type SeedRatioMode int64

const (
	// SeedRatioModeGlobal represents the use of the global ratio for a torrent
	SeedRatioModeGlobal SeedRatioMode = 0
	// SeedRatioModeCustom represents the use of a custom ratio for a torrent
	SeedRatioModeCustom SeedRatioMode = 1
	// SeedRatioModeNoRatio represents the absence of ratio for a torrent
	SeedRatioModeNoRatio SeedRatioMode = 2
)

func (srm SeedRatioMode) String() string {
	switch srm {
	case SeedRatioModeGlobal:
		return "global"
	case SeedRatioModeCustom:
		return "custom"
	case SeedRatioModeNoRatio:
		return "no ratio"
	default:
		return "<unknown>"
	}
}

// GoString implements the GoStringer interface from the stdlib fmt package
func (srm SeedRatioMode) GoString() string {
	switch srm {
	case SeedRatioModeGlobal:
		return fmt.Sprintf("global (%d)", srm)
	case SeedRatioModeCustom:
		return fmt.Sprintf("custom (%d)", srm)
	case SeedRatioModeNoRatio:
		return fmt.Sprintf("no ratio (%d)", srm)
	default:
		return fmt.Sprintf("<unknown> (%d)", srm)
	}
}

type Tracker struct {
	Announce string `json:"announce"`
	ID       int64  `json:"id"`
	Scrape   string `json:"scrape"`
	SiteName string `json:"sitename"`
	Tier     int64  `json:"tier"`
}

// TrackerStats represent the extended data of a torrent's tracker.
type TrackerStats struct {
	Announce              string    `json:"announce"`
	AnnounceState         int64     `json:"announceState"`
	DownloadCount         int64     `json:"downloadCount"`
	HasAnnounced          bool      `json:"hasAnnounced"`
	HasScraped            bool      `json:"hasScraped"`
	Host                  string    `json:"host"`
	ID                    int64     `json:"id"`
	IsBackup              bool      `json:"isBackup"`
	LastAnnouncePeerCount int64     `json:"lastAnnouncePeerCount"`
	LastAnnounceResult    string    `json:"lastAnnounceResult"`
	LastAnnounceStartTime time.Time `json:"-"`
	LastAnnounceSucceeded bool      `json:"lastAnnounceSucceeded"`
	LastAnnounceTime      time.Time `json:"-"`
	LastAnnounceTimedOut  bool      `json:"lastAnnounceTimedOut"`
	LastScrapeResult      string    `json:"lastScrapeResult"`
	LastScrapeStartTime   time.Time `json:"-"`
	LastScrapeSucceeded   bool      `json:"lastScrapeSucceeded"`
	LastScrapeTime        time.Time `json:"-"`
	LastScrapeTimedOut    bool      `json:"-"` // should be boolean but number. Will be converter in UnmarshalJSON
	LeecherCount          int64     `json:"leecherCount"`
	NextAnnounceTime      time.Time `json:"-"`
	NextScrapeTime        time.Time `json:"-"`
	Scrape                string    `json:"scrape"`
	ScrapeState           int64     `json:"scrapeState"`
	SiteName              string    `json:"sitename"`
	SeederCount           int64     `json:"seederCount"`
	Tier                  int64     `json:"tier"`
}

type TorrentStatus int64

const (
	// TorrentStatusStopped represents a stopped torrent
	TorrentStatusStopped TorrentStatus = 0
	// TorrentStatusCheckWait represents a torrent queued for files checking
	TorrentStatusCheckWait TorrentStatus = 1
	// TorrentStatusCheck represents a torrent which files are currently checked
	TorrentStatusCheck TorrentStatus = 2
	// TorrentStatusDownloadWait represents a torrent queue to download
	TorrentStatusDownloadWait TorrentStatus = 3
	// TorrentStatusDownload represents a torrent currently downloading
	TorrentStatusDownload TorrentStatus = 4
	// TorrentStatusSeedWait represents a torrent queued to seed
	TorrentStatusSeedWait TorrentStatus = 5
	// TorrentStatusSeed represents a torrent currently seeding
	TorrentStatusSeed TorrentStatus = 6
	// TorrentStatusIsolated represents a torrent which can't find peers
	TorrentStatusIsolated TorrentStatus = 7
)

func (status TorrentStatus) String() string {
	switch status {
	case TorrentStatusStopped:
		return "stopped"
	case TorrentStatusCheckWait:
		return "waiting to check files"
	case TorrentStatusCheck:
		return "checking files"
	case TorrentStatusDownloadWait:
		return "waiting to download"
	case TorrentStatusDownload:
		return "downloading"
	case TorrentStatusSeedWait:
		return "waiting to seed"
	case TorrentStatusSeed:
		return "seeding"
	case TorrentStatusIsolated:
		return "can't find peers"
	default:
		return "<unknown>"
	}
}
