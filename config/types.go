package config

import "gopkg.in/yaml.v3"

type CacheType uint
type LyricsPlacement uint

const (
	CACHE_NONE CacheType = iota
	CACHE_LIKED_ONLY
	CACHE_ALL
)
const (
	LYRICS_PLACEMENT_ABOVE LyricsPlacement = iota
	LYRICS_PLACEMENT_BELOW
)

var cacheValueToEnum = map[string]CacheType{
	"disable": CACHE_NONE,
	"false":   CACHE_NONE,
	"none":    CACHE_NONE,
	"off":     CACHE_NONE,
	"likes":   CACHE_LIKED_ONLY,
	"liked":   CACHE_LIKED_ONLY,
	"all":     CACHE_ALL,
}

var cacheEnumToValue = map[CacheType]string{
	CACHE_NONE:       "none",
	CACHE_LIKED_ONLY: "likes",
	CACHE_ALL:        "all",
}

var lyricsPlacementValueToEnum = map[string]LyricsPlacement{
	"below": LYRICS_PLACEMENT_BELOW,
	"down":  LYRICS_PLACEMENT_BELOW,
	"above": LYRICS_PLACEMENT_ABOVE,
	"up":    LYRICS_PLACEMENT_ABOVE,
}

var lyricsPlacementEnumToValue = map[LyricsPlacement]string{
	LYRICS_PLACEMENT_BELOW: "below",
	LYRICS_PLACEMENT_ABOVE: "above",
}

func (t *CacheType) UnmarshalYAML(value *yaml.Node) error {
	*t = cacheValueToEnum[value.Value]
	return nil
}

func (t CacheType) MarshalYAML() (interface{}, error) {
	if t > CACHE_ALL {
		t = CACHE_NONE
	}
	return cacheEnumToValue[t], nil
}

func (t *LyricsPlacement) UnmarshalYAML(value *yaml.Node) error {
	*t = lyricsPlacementValueToEnum[value.Value]
	return nil
}

func (t LyricsPlacement) MarshalYAML() (interface{}, error) {
	if t > LYRICS_PLACEMENT_BELOW {
		t = LYRICS_PLACEMENT_ABOVE
	}
	return lyricsPlacementEnumToValue[t], nil
}

type Controls struct {
	// Main control
	Quit        *Key `yaml:"quit"`
	Apply       *Key `yaml:"apply"`
	Cancel      *Key `yaml:"cancel"`
	CursorUp    *Key `yaml:"cursor-up"`
	CursorDown  *Key `yaml:"cursor-down"`
	ShowAllKeys *Key `yaml:"show-all-kyes"`
	// Playlists control
	PlaylistsUp     *Key `yaml:"playlists-up"`
	PlaylistsDown   *Key `yaml:"playlists-down"`
	PlaylistsRename *Key `yaml:"playlists-rename"`
	// Track list control
	TracksLike               *Key `yaml:"tracks-like"`
	TracksAddToPlaylist      *Key `yaml:"tracks-add-to-playlist"`
	TracksRemoveFromPlaylist *Key `yaml:"tracks-remove-from-playlist"`
	TracksShare              *Key `yaml:"tracks-share"`
	TracksShuffle            *Key `yaml:"tracks-shuffle"`
	TracksSearch             *Key `yaml:"tracks-search"`
	// Player control
	PlayerPause          *Key `yaml:"player-pause"`
	PlayerNext           *Key `yaml:"player-next"`
	PlayerPrevious       *Key `yaml:"player-previous"`
	PlayerRewindForward  *Key `yaml:"player-rewind-forward"`
	PlayerRewindBackward *Key `yaml:"player-rewind-backward"`
	PlayerLike           *Key `yaml:"player-like"`
	PlayerCache          *Key `yaml:"player-cache"`
	PlayerVolUp          *Key `yaml:"player-vol-up"`
	PlayerVolDown        *Key `yaml:"player-vol-donw"`
	PlayerToggleLyrics   *Key `yaml:"player-toggle-lyrics"`
}

type Search struct {
	Artists   bool `yaml:"artists"`
	Albums    bool `yaml:"albums"`
	Playlists bool `yaml:"playlists"`
}

type Config struct {
	Token           string          `yaml:"token"`
	BufferSize      float64         `yaml:"buffer-size-ms"`
	RewindDuration  float64         `yaml:"rewind-duration-s"`
	Volume          float64         `yaml:"volume"`
	VolumeStep      float64         `yaml:"volume-step"`
	ShowLyrics      bool            `yaml:"show-lyrics"`
	LyricsPlacement LyricsPlacement `yaml:"lyrics-placement"`
	CacheTracks     CacheType       `yaml:"cache-tracks"`
	CacheDir        string          `yaml:"cache-dir"`
	Search          *Search         `yaml:"search"`
	Controls        *Controls       `yaml:"controls"`
}

var defaultConfig = Config{
	BufferSize:      80,
	RewindDuration:  5,
	Volume:          0.5,
	VolumeStep:      0.05,
	ShowLyrics:      false,
	LyricsPlacement: LYRICS_PLACEMENT_ABOVE,
	CacheTracks:     CACHE_LIKED_ONLY,
	CacheDir:        "",
	Search: &Search{
		Artists:   true,
		Albums:    false,
		Playlists: false,
	},
	Controls: &Controls{
		Quit:                     NewKey("ctrl+q,ctrl+c"),
		Apply:                    NewKey("enter"),
		Cancel:                   NewKey("esc"),
		CursorUp:                 NewKey("up"),
		CursorDown:               NewKey("down"),
		ShowAllKeys:              NewKey("?"),
		PlaylistsUp:              NewKey("ctrl+up"),
		PlaylistsDown:            NewKey("ctrl+down"),
		PlaylistsRename:          NewKey("ctrl+r"),
		TracksLike:               NewKey("l"),
		TracksAddToPlaylist:      NewKey("a"),
		TracksRemoveFromPlaylist: NewKey("ctrl+a"),
		TracksSearch:             NewKey("ctrl+f"),
		TracksShuffle:            NewKey("ctrl+x"),
		TracksShare:              NewKey("ctrl+s"),
		PlayerPause:              NewKey("space"),
		PlayerNext:               NewKey("right"),
		PlayerPrevious:           NewKey("left"),
		PlayerRewindForward:      NewKey("ctrl+right"),
		PlayerRewindBackward:     NewKey("ctrl+left"),
		PlayerLike:               NewKey("L"),
		PlayerToggleLyrics:       NewKey("t"),
		PlayerCache:              NewKey("S"),
		PlayerVolUp:              NewKey("+,="),
		PlayerVolDown:            NewKey("-"),
	},
}

const ConfigPath = "yamusic-tui"
