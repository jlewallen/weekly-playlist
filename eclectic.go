package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type PlaylistTrack struct {
	AffiliateLinkiPhone  string    `json:"affiliateLinkiPhone"`
	ProgramStart         string    `json:"program_start"`
	AffiliateLinkSpotify string    `json:"affiliateLinkSpotify"`
	PerformanceStart     string    `json:"performance_start"`
	ArtistLink           string    `json:"artist_link"`
	PlayId               int64     `json:"play_id"`
	Offset               int64     `json:"offset"`
	ProgramId            string    `json:"program_id"`
	Datetime             time.Time `json:"datetime"`
	ProgramEnd           string    `json:"program_end"`
	AffiliateLinkRdio    string    `json:"affiliateLinkRdio"`
	AlbumImage           string    `json:"albumImage"`
	Year                 int32     `json:"year"`
	Date                 DateOnly  `json:"date"`
	FeatureUrl           string    `json:"feature_url"`
	ProgramTitle         string    `json:"program_title"`
	ListenLink           string    `json:"listen_link"`
	FeatureTitle         string    `json:"feature_title"`
	AlbumImageLarge      string    `json:"albumImageLarge"`
	Album                string    `json:"album"`
	Guest                string    `json:"guest"`
	Title                string    `json:"title"`
	Artist               string    `json:"artist"`
	Host                 string    `json:"host"`
	Comments             string    `json:"comments"`
	Label                string    `json:"label"`
	AffiliateLinkiTunes  string    `json:"affiliateLinkiTunes"`
	Live                 string    `json:"live"`
	Artist_url           string    `json:"artist_url"`
	AffiliateLinkAmazon  string    `json:"affiliateLinkAmazon"`
	Time                 string    `json:"time"`
	Action               string    `json:"action"`
	Credits              string    `json:"credits"`
	Channel              string    `json:"channel"`
	Location             string    `json:"location"`
}

type PlaylistTracks struct {
	Show   time.Time
	Search string
	Tracks *[]PlaylistTrack
}

func NewPlaylistTracks(show time.Time) (*PlaylistTracks, error) {
	search := show.Format("2006/01/02?time=15:04")
	url := "http://tracklist-api.kcrw.com/Music/date/" + search
	if false {
		log.Printf("%s", url)
	}
	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	var tracks []PlaylistTrack

	if err := json.NewDecoder(r.Body).Decode(&tracks); err != nil {
		return nil, err
	}

	return &PlaylistTracks{
		Show:   show,
		Search: search,
		Tracks: &tracks,
	}, nil
}

type Eclectic24 struct {
	Show time.Time
}

func NewEclectic24() *Eclectic24 {
	now := time.Now()
	h := (now.Hour() / 3) * 3
	show := time.Date(now.Year(), now.Month(), now.Day(), h, 0, 0, 0, now.Location())

	return &Eclectic24{
		Show: show,
	}
}

func (e *Eclectic24) NextShow() {
	e.Show = e.Show.Add(3 * time.Hour)
}

func (e *Eclectic24) PreviousShow() {
	e.Show = e.Show.Add(-3 * time.Hour)
}

/*
func main() {
	e := NewEclectic24()

	pl, err := NewPlaylistTracks(e.Show)
	if err != nil {
		log.Fatalf("Error %v", err)
	}
	for _, track := range *pl.Tracks {
		fmt.Printf("%v %s %s\n", track.Artist, track.Title, track.AffiliateLinkSpotify)
	}
	fmt.Printf("%d\n", len(*pl.Tracks))

	e.PreviousShow()

	pl, err = NewPlaylistTracks(e.Show)
	if err != nil {
		log.Fatalf("Error %v", err)
	}
	for _, track := range *pl.Tracks {
		fmt.Printf("%v %s %s\n", track.Artist, track.Title, track.AffiliateLinkSpotify)
	}
	fmt.Printf("%d\n", len(*pl.Tracks))
}
*/
