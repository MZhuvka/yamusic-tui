package tracker

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"io"
	"math"
	"strings"
	"time"

	"github.com/BourgeoisBear/rasterm"
	"github.com/BurntSushi/graphics-go/graphics"
	"github.com/BurntSushi/graphics-go/graphics/interp"
	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/stream"
	"github.com/dece2183/yamusic-tui/ui/helpers"
	"github.com/dece2183/yamusic-tui/ui/model"
	"github.com/dece2183/yamusic-tui/ui/style"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/ebitengine/oto/v3"
)

type Control uint

const (
	PLAY Control = iota
	PAUSE
	STOP
	NEXT
	PREV
	LIKE
	REWIND
	VOLUME
	CACHE_TRACK
	BUFFERING_COMPLETE
)

type ProgressControl float64

func (p ProgressControl) Value() float64 {
	return float64(p)
}

var rewindAmount = time.Duration(config.Current.RewindDuration) * time.Second

type Model struct {
	width    int
	cover    string
	track    api.Track
	progress progress.Model
	help     help.Model

	volume        float64
	playerContext *oto.Context
	player        *oto.Player
	trackWrapper  *readWrapper

	program  *tea.Program
	likesMap *map[string]bool
}

func New(p *tea.Program, likesMap *map[string]bool) *Model {
	m := &Model{
		program:  p,
		likesMap: likesMap,
		progress: progress.New(progress.WithSolidFill(string(style.AccentColor))),
		help:     help.New(),
		volume:   config.Current.Volume,
	}

	m.progress.ShowPercentage = false
	m.progress.Empty = m.progress.Full
	m.progress.EmptyColor = string(style.BackgroundColor)

	m.trackWrapper = &readWrapper{program: m.program}

	op := &oto.NewContextOptions{
		SampleRate:   44100,
		ChannelCount: 2,
		BufferSize:   time.Millisecond * time.Duration(config.Current.BufferSize),
		Format:       oto.FormatSignedInt16LE,
	}

	var err error
	var readyChan chan struct{}
	m.playerContext, readyChan, err = oto.NewContext(op)
	if err != nil {
		model.PrettyExit(err, 12)
	}
	<-readyChan

	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) View() string {
	var playButton string
	if m.IsPlaying() {
		playButton = style.ActiveButtonStyle.Padding(0, 1).Margin(0).Render(style.IconPlay)
	} else {
		playButton = style.ActiveButtonStyle.Padding(0, 1).Margin(0).Render(style.IconStop)
	}

	var trackTitle string
	if m.help.ShowAll {
		trackTitle = lipgloss.JoinVertical(lipgloss.Left, "")
	} else {
		if m.track.Available {
			trackTitle = style.TrackTitleStyle.Render(m.track.Title)
		} else {
			trackTitle = style.TrackTitleStyle.Strikethrough(true).Render(m.track.Title)
		}

		trackVersion := style.TrackVersionStyle.Render(" " + m.track.Version)
		trackTitle = lipgloss.JoinHorizontal(lipgloss.Top, trackTitle, trackVersion)

		durTotal := time.Millisecond * time.Duration(m.track.DurationMs)
		durEllapsed := time.Millisecond * time.Duration(float64(m.track.DurationMs)*m.progress.Percent())
		trackTime := style.TrackVersionStyle.Render(fmt.Sprintf("%02d:%02d/%02d:%02d",
			int(durEllapsed.Minutes()),
			int(durEllapsed.Seconds())%60,
			int(durTotal.Minutes()),
			int(durTotal.Seconds())%60,
		))

		var trackLike string
		if (*m.likesMap)[m.track.Id] {
			trackLike = style.IconLiked + " "
		} else {
			trackLike = style.IconNotLiked + " "
		}

		trackAddInfo := style.TrackAddInfoStyle.Render(trackLike + trackTime)
		addInfoLen := lipgloss.Width(trackAddInfo)
		maxLen := m.Width() - addInfoLen - 4 - 14
		stl := lipgloss.NewStyle().MaxWidth(maxLen - 1)

		trackTitleLen := lipgloss.Width(trackTitle)
		if trackTitleLen > maxLen {
			trackTitle = stl.Render(trackTitle) + "…"
		} else if trackTitleLen < maxLen {
			trackTitle += strings.Repeat(" ", maxLen-trackTitleLen)
		}

		trackArtist := style.TrackArtistStyle.Render(helpers.ArtistList(m.track.Artists))
		trackArtistLen := lipgloss.Width(trackArtist)
		if trackArtistLen > maxLen {
			trackArtist = stl.Render(trackArtist) + "…"
		} else if trackArtistLen < maxLen {
			trackArtist += strings.Repeat(" ", maxLen-trackArtistLen)
		}

		trackTitle = lipgloss.NewStyle().Width(m.width - lipgloss.Width(trackAddInfo) - 4 - 14).Render(trackTitle)
		trackTitle = lipgloss.JoinHorizontal(lipgloss.Top, trackTitle, trackAddInfo)

		trackTitle = lipgloss.JoinVertical(lipgloss.Left, trackTitle, trackArtist, "")
	}

	tracker := style.TrackProgressStyle.Render(m.progress.View())
	tracker = lipgloss.JoinHorizontal(lipgloss.Top, playButton, tracker)
	tracker = lipgloss.JoinVertical(lipgloss.Left, tracker, trackTitle, m.help.View(helpMap))

	if len(m.cover) > 0 {
		tracker = lipgloss.JoinHorizontal(lipgloss.Top, style.TrackCoverStyle.Render(m.cover), tracker)
		// tracker = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.Place(14, 6, 0, 0, m.cover), tracker)
		// tracker = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.PlaceHorizontal(14, 0, m.cover), tracker)
	}

	return style.TrackBoxStyle.Width(m.width).Render(tracker)
}

func (m *Model) Update(message tea.Msg) (*Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := message.(type) {
	case tea.KeyMsg:
		controls := config.Current.Controls
		keypress := msg.String()

		switch {
		case controls.ShowAllKeys.Contains(keypress):
			m.help.ShowAll = !m.help.ShowAll

		case controls.PlayerPause.Contains(keypress):
			if m.player == nil {
				break
			}
			if m.player.IsPlaying() {
				m.Pause()
				cmds = append(cmds, model.Cmd(PAUSE))
			} else {
				m.Play()
				cmds = append(cmds, model.Cmd(PLAY))
			}

		case controls.PlayerRewindForward.Contains(keypress):
			m.Rewind(rewindAmount)
			cmds = append(cmds, model.Cmd(REWIND))

		case controls.PlayerRewindBackward.Contains(keypress):
			m.Rewind(-rewindAmount)
			cmds = append(cmds, model.Cmd(REWIND))

		case controls.PlayerNext.Contains(keypress):
			cmds = append(cmds, model.Cmd(NEXT))

		case controls.PlayerPrevious.Contains(keypress):
			cmds = append(cmds, model.Cmd(PREV))

		case controls.PlayerLike.Contains(keypress):
			cmds = append(cmds, model.Cmd(LIKE))

		case controls.PlayerCache.Contains(keypress):
			if !m.IsStoped() {
				m.trackWrapper.trackBuffer.BufferAll()
				cmds = append(cmds, model.Cmd(CACHE_TRACK))
			}

		case controls.PlayerVolUp.Contains(keypress):
			m.SetVolume(m.volume + config.Current.VolumeStep)
			config.Current.Volume = m.volume
			config.Save()
			cmds = append(cmds, model.Cmd(VOLUME))

		case controls.PlayerVolDown.Contains(keypress):
			m.SetVolume(m.volume - config.Current.VolumeStep)
			config.Current.Volume = m.volume
			config.Save()
			cmds = append(cmds, model.Cmd(VOLUME))

		}

	// player control update
	case Control:
		switch msg {
		case PLAY:
			m.Play()
		case PAUSE:
			m.Pause()
		case STOP:
			m.Stop()
		}

	// track progress update
	case ProgressControl:
		cmd = m.progress.SetPercent(msg.Value())
		cmds = append(cmds, cmd)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) SetWidth(width int) {
	m.width = width
	m.progress.Width = width - 9 - 14
	m.help.Width = width - 8 - 14
}

func (m *Model) Width() int {
	return m.width
}

func (m *Model) Progress() float64 {
	return m.progress.Percent()
}

func (m *Model) Position() time.Duration {
	return time.Duration(float64(m.track.DurationMs)*m.trackWrapper.Progress()) * time.Millisecond
}

func (m *Model) SetVolume(v float64) {
	m.volume = v

	if m.volume < 0 {
		m.volume = 0
	} else if m.volume > 1 {
		m.volume = 1
	}

	if m.player != nil {
		m.player.SetVolume(v)
	}
}

func (m *Model) Volume() float64 {
	return m.volume
}

func (m *Model) StartTrack(track *api.Track, cover image.Image, reader *stream.BufferedStream) {
	if m.player != nil {
		m.Stop()
	}

	const (
		chWidth  = 12
		chHeight = 6
		pixHor   = 10
		pixVer   = 20
	)

	if cover != nil {
		coverScaled := image.NewPaletted(image.Rect(0, 0, chWidth*pixHor, chHeight*pixVer), palette.Plan9)
		graphics.I.Scale(0.62, 0.62).TransformCenter(coverScaled, cover, interp.Bilinear)

		str := &strings.Builder{}
		r := image.Rect(0, 0, chWidth*pixHor, pixVer)
		coverSlice := image.NewPaletted(r, palette.Plan9)
		for i := 0; i < chHeight; i++ {
			draw.Draw(coverSlice, r, coverScaled, image.Pt(0, i*pixVer), draw.Src)
			str.WriteString(strings.Repeat("?", chWidth))
			str.WriteString(ansi.CUB(chWidth))
			rasterm.SixelWriteImage(str, coverSlice)
			str.WriteString(ansi.CUF(chWidth))
			str.WriteRune('\n')
		}

		m.cover = str.String()
	}

	m.track = *track
	m.trackWrapper.NewReader(reader)
	m.player = m.playerContext.NewPlayer(m.trackWrapper)
	m.player.SetVolume(m.volume)
	m.player.Play()
}

func (m *Model) Stop() {
	if m.player == nil {
		return
	}

	if m.player.IsPlaying() {
		m.player.Pause()
	}

	m.trackWrapper.Close()
	m.player.Close()
	m.player = nil
}

func (m *Model) IsPlaying() bool {
	return m.player != nil && m.trackWrapper.trackBuffer != nil && m.player.IsPlaying()
}

func (m *Model) IsStoped() bool {
	return m.player == nil || m.trackWrapper.trackBuffer == nil
}

func (m *Model) CurrentTrack() *api.Track {
	return &m.track
}

func (m *Model) Play() {
	if m.player == nil || m.trackWrapper.trackBuffer == nil {
		return
	}
	if m.player.IsPlaying() {
		return
	}
	m.player.Play()
}

func (m *Model) Pause() {
	if m.player == nil || m.trackWrapper.trackBuffer == nil {
		return
	}
	if !m.player.IsPlaying() {
		return
	}
	m.player.Pause()
}

func (m *Model) Rewind(amount time.Duration) {
	if m.player == nil || m.trackWrapper == nil {
		go m.program.Send(STOP)
		return
	}

	amountMs := amount.Milliseconds()
	currentPos := int64(float64(m.trackWrapper.Length()) * m.trackWrapper.Progress())
	byteOffset := int64(math.Round((float64(m.trackWrapper.Length()) / float64(m.track.DurationMs)) * float64(amountMs)))

	// align position by 4 bytes
	currentPos += byteOffset
	currentPos += currentPos % 4

	if currentPos <= 0 {
		m.player.Seek(0, io.SeekStart)
	} else if currentPos >= m.trackWrapper.Length() {
		m.player.Seek(0, io.SeekEnd)
	} else {
		m.player.Seek(currentPos, io.SeekStart)
	}
}

func (m *Model) SetPos(pos time.Duration) {
	if m.player == nil || m.trackWrapper == nil {
		go m.program.Send(STOP)
		return
	}

	posMs := pos.Milliseconds()
	byteOffset := int64(math.Round((float64(m.trackWrapper.Length()) / float64(m.track.DurationMs)) * float64(posMs)))

	// align position by 4 bytes
	byteOffset += byteOffset % 4
	m.player.Seek(byteOffset, io.SeekStart)
}

func (m *Model) TrackBuffer() *stream.BufferedStream {
	return m.trackWrapper.trackBuffer
}
