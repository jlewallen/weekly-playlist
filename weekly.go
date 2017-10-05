package main

import (
	"bytes"
	"flag"
	mapset "github.com/deckarep/golang-set"
	fb "github.com/huandu/facebook"
	"github.com/zmb3/spotify"
	"io"
	"log"
    "fmt"
	"os"
	"strings"
	"time"
    "net/url"
    "regexp"
)

type Event struct {
	Name     string
	Location string
	Artists  *ArtistGuess
}

type ArtistResolver struct {
	artistCache    map[string]*spotify.FullArtist
	spotifyArtists map[string]*spotify.FullArtist
}

func ProcessVenue(facebookSession *fb.Session, venueId string) (events []Event) {
	venueInfo, _ := GetVenueInformation(facebookSession, venueId)
	log.Println(venueInfo.Name)

	facebookEvents, _ := GetUpcomingFacebookEvents(facebookSession, venueId)
	for _, event := range facebookEvents {
		artists := GuessArtistsForEvent(event.Name)
		events = append(events, Event{
			Name:     event.Name,
			Location: "",
			Artists:  artists,
		})
	}

	return
}

func NewArtistResolver() (resolver *ArtistResolver) {
	resolver = new(ArtistResolver)
	resolver.artistCache = make(map[string]*spotify.FullArtist)
	resolver.spotifyArtists = make(map[string]*spotify.FullArtist)

	return resolver
}

func (resolver *ArtistResolver) SearchWithRetry(spotifyClient *spotify.Client, st spotify.SearchType, term string) (sr *spotify.SearchResult, err error) {
	country := "US"
	for {
		sr, err = spotifyClient.SearchOpt(term, st, &spotify.Options{Country: &country})
		if err != nil {
			if !strings.Contains(err.Error(), "rate") {
				break
			}

			log.Printf("Throttled...")
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	return
}

func (resolver *ArtistResolver) GetSpotifyArtistsForGuess(spotifyClient *spotify.Client, depth int, artist *ArtistGuess) {
	log.Printf("      [%-4s]%s%s\n", artist.Step, strings.Repeat("  ", depth), artist.Name)

	anyFound := false

	if resolver.artistCache[artist.Name] == nil {
		found, err := resolver.SearchWithRetry(spotifyClient, spotify.SearchTypeArtist, artist.Name)
		if err != nil {
			log.Printf("Error:", err)
		} else {
			if found.Artists != nil {
				for _, item := range found.Artists.Artists {
					if strings.ToLower(item.Name) == strings.ToLower(artist.Name) {
						log.Printf("      [%-4s]%s%s\n", "****", strings.Repeat("  ", depth), item.Name)
						resolver.artistCache[artist.Name] = &item
						resolver.spotifyArtists[artist.Name] = &item
						anyFound = true
						break
					}
				}
			}
		}
	} else {
		resolver.spotifyArtists[artist.Name] = resolver.artistCache[artist.Name]
	}

	if !anyFound {
		for _, child := range artist.Children {
			resolver.GetSpotifyArtistsForGuess(spotifyClient, depth+1, child)
		}
	}
}

func (resolver *ArtistResolver) GetSpotifyArtists(spotifyClient *spotify.Client, event Event) (spotifyArtists map[string]*spotify.FullArtist) {
	resolver.spotifyArtists = make(map[string]*spotify.FullArtist)

	resolver.GetSpotifyArtistsForGuess(spotifyClient, 0, event.Artists)

	spotifyArtists = resolver.spotifyArtists

	return
}

func (resolver *ArtistResolver) GetArtistTracks(spotifyClient *spotify.Client, artist spotify.FullArtist) (tracks []spotify.FullTrack) {
	topTracks, _ := spotifyClient.GetArtistsTopTracks(artist.ID, "US")
	if len(topTracks) > 3 {
		tracks = topTracks[:3]
	} else {
		tracks = topTracks
	}
	return
}

func GetPlaylistByTitle(spotifyClient *spotify.Client, name string) (*spotify.SimplePlaylist, error) {
	limit := 20
	offset := 0
	options := spotify.Options{Limit: &limit, Offset: &offset}
	for {
        playlists, err := spotifyClient.GetPlaylistsForUserOpt("jlewalle", &options)
        if err != nil {
            return nil, err
        }

        for _, iter := range playlists.Playlists {
            // log.Printf("'%s' == '%s'", iter.Name, name)
            if strings.EqualFold(iter.Name, name) {
                return &iter, nil
            }
        }

		if len(playlists.Playlists) < *options.Limit {
			break
		}

		offset := *options.Limit + *options.Offset
		options.Offset = &offset
	}

	return nil, nil
}

func GetPlaylistTracks(spotifyClient *spotify.Client, userId string, id spotify.ID) (allTracks []spotify.PlaylistTrack, err error) {
	limit := 100
	offset := 0
	options := spotify.Options{Limit: &limit, Offset: &offset}
	for {
		tracks, spotifyErr := spotifyClient.GetPlaylistTracksOpt(userId, id, &options, "")
		if spotifyErr != nil {
			err = spotifyErr
			return
		}

		allTracks = append(allTracks, tracks.Tracks...)

		if len(tracks.Tracks) < *options.Limit {
			break
		}

		offset := *options.Limit + *options.Offset
		options.Offset = &offset
	}

	return
}

func GetFullTracks(tracks []spotify.PlaylistTrack) (fullTracks []spotify.FullTrack) {
	for _, track := range tracks {
		fullTracks = append(fullTracks, track.Track)
	}

	return
}

func GetTrackIds(tracks []spotify.FullTrack) (ids []spotify.ID) {
	for _, track := range tracks {
		ids = append(ids, track.ID)
	}

	return
}

func ToSpotifyIds(ids []interface{}) (ifaces []spotify.ID) {
	for _, id := range ids {
		ifaces = append(ifaces, id.(spotify.ID))
	}
	return
}

func MapIds(ids []spotify.ID) (ifaces []interface{}) {
	for _, id := range ids {
		ifaces = append(ifaces, id)
	}
	return
}

type Options struct {
	GuessOnly    bool
	EclecticOnly bool
	RegionsFile  string
}

var nonLetters = regexp.MustCompile("[\\W\\D]")

type PlaylistUpdate struct {
    idsBefore mapset.Set
    idsAfter []spotify.ID
}

func NewPlaylistUpdate(idsBefore []spotify.ID) (*PlaylistUpdate) {
    return &PlaylistUpdate{
        idsBefore: mapset.NewSetFromSlice(MapIds(idsBefore)),
        idsAfter: make([]spotify.ID, 0),
    }
}

func (pu *PlaylistUpdate) AddTrack(id spotify.ID) {
    pu.idsAfter = append(pu.idsAfter, id)
}

func (pu *PlaylistUpdate) GetIdsToRemove() []spotify.ID {
    afterSet := mapset.NewSetFromSlice(MapIds(pu.idsAfter))
    idsToRemove := pu.idsBefore.Difference(afterSet)
    return ToSpotifyIds(idsToRemove.ToSlice())
}

func (pu *PlaylistUpdate) GetIdsToAdd() []spotify.ID {
    ids := make([]spotify.ID, 0)
    for _, id := range pu.idsAfter {
        if !pu.idsBefore.Contains(id) {
            ids = append(ids, id)
        }
    }
    return ids
}

func (pu *PlaylistUpdate) MergeBeforeAndToAdd() {
    for _, id := range pu.idsAfter {
        pu.idsBefore.Add(id)
    }
}

func LooselyEqual(a string, b string) bool {
    newA := nonLetters.ReplaceAllString(a, "")
    newB := nonLetters.ReplaceAllString(b, "")
    return strings.EqualFold(newA, newB)
}

func SameTrack(t PlaylistTrack, st spotify.FullTrack) bool {
    if !LooselyEqual(t.Title, st.Name) {
        log.Printf("   '%s' != '%s'", t.Title, st.Name)
        return false
    }
    return true
}

func SelectTrack(t PlaylistTrack, tracks []spotify.FullTrack) *spotify.FullTrack {
    for _, r := range tracks {
        if SameTrack(t, r) {
            log.Printf("   %s: %s - %s", r.ID, r.Artists[0].Name, r.Name)
            return &r
        }
    }
    return nil
}

func GetLastSunday(t time.Time) time.Time {
	i := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
    for {
        if i.Weekday() == time.Sunday {
            return i
        }
        i = i.Add(-24 * time.Hour)
    }
}

func UpdateEclectic(spotifyClient *spotify.Client, e *Eclectic24) (error) {
    week := GetLastSunday(e.Show)
    name := fmt.Sprintf("mbe %s", week.Format("06/01/02"))

    log.Printf("Generating %v", name)
    playlist, err := GetPlaylistByTitle(spotifyClient, name)
    if err != nil {
        log.Fatalf("%v", err)
    }

    var playlistId spotify.ID
    if playlist == nil {
        log.Printf("Creating %v", name)

        created, err := spotifyClient.CreatePlaylistForUser("jlewalle", name, true)
        if err != nil {
            log.Fatalf("%v", err)
        }

        playlistId = created.ID
    } else {
        playlistId = playlist.ID
    }

    tracksBefore, _ := GetPlaylistTracks(spotifyClient, "jlewalle", playlistId)
    idsBefore := GetTrackIds(GetFullTracks(tracksBefore))
    update := NewPlaylistUpdate(idsBefore)

    log.Printf("Before: %d", len(idsBefore))

	ar := NewArtistResolver()

    for {
        pl, err := NewPlaylistTracks(e.Show)
        if err != nil {
            return err
        }

        fmt.Printf("%v %v\n", len(*pl.Tracks), e.Show)

        for _, track := range *pl.Tracks {
            if track.AffiliateLinkSpotify != "" {

                query, _ := url.QueryUnescape(track.AffiliateLinkSpotify)
                query = strings.Replace(query, "spotify:search:", "", 1) 

                log.Printf("%v - %s", track.Artist, track.Title)

                f, err := ar.SearchWithRetry(spotifyClient, spotify.SearchTypeTrack, query)
                if err != nil {
                    return err
                }

                sel := SelectTrack(track, f.Tracks.Tracks) 
                if sel != nil {
                    update.AddTrack(sel.ID)
                }
            }
        }

        idsToAdd := update.GetIdsToAdd()
        log.Printf("Adding %d tracks to %s", len(idsToAdd), playlistId)

        for i := 0; i < len(idsToAdd); i += 50 {
            batch := idsToAdd[i:min(i+50, len(idsToAdd))]
            spotifyClient.AddTracksToPlaylist("jlewalle", playlistId, batch...)
        }

        update.MergeBeforeAndToAdd()

        again := len(idsToAdd) > 0 && len(*pl.Tracks) > 0
        if !again {
            log.Printf("No new tracks, done")
            return nil
        }

        e.PreviousShow()

        if GetLastSunday(e.Show) != week {
            log.Printf("Got to the beginning of the week, done")
            return nil
        }
    }

	return nil
}

func main() {
	var options Options

	flag.BoolVar(&options.GuessOnly, "guess-only", false, "test guessing code only")
	flag.BoolVar(&options.EclecticOnly, "eclectic-only", false, "only update mbe playlist")
	flag.StringVar(&options.RegionsFile, "regions-file", "regions.json", "json regions file to use")

	flag.Parse()

	logFile, err := os.OpenFile("weekly.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer logFile.Close()

	buffer := new(bytes.Buffer)

	multi := io.MultiWriter(logFile, buffer, os.Stdout)

	log.SetOutput(multi)

	spotifyClient, _ := AuthenticateSpotify()
	facebookSession, _ := AuthenticateFacebook()
	regions := LoadRegions(options.RegionsFile)
	artistsResolver := NewArtistResolver()

	if !options.EclecticOnly {
		for _, region := range regions {
			tracksAfter := []spotify.FullTrack{}

			for _, venueId := range region.VenueIds {
				for _, event := range ProcessVenue(facebookSession, venueId) {
					foundTracks := false
					log.Printf("   '%s'\n", event.Name)
					artists := artistsResolver.GetSpotifyArtists(spotifyClient, event)
					for _, value := range artists {
						artistTracks := artistsResolver.GetArtistTracks(spotifyClient, *value)
						tracksAfter = append(tracksAfter, artistTracks...)
						log.Printf("   %d tracks '%s'\n", len(artistTracks), value.Name)
						foundTracks = true
					}
					if !foundTracks {
						log.Printf("   NO TRACKS")
					}

					log.Printf("")
				}
			}

			if !options.GuessOnly {
				idsAfter := GetTrackIds(tracksAfter)

				playlist, err := GetPlaylistByTitle(spotifyClient, region.Region+" weekly")
				if err != nil {
					log.Printf("%v\n", err)
					os.Exit(1)
				}
				tracksBefore, _ := GetPlaylistTracks(spotifyClient, "jlewalle", playlist.ID)
				idsBefore := GetTrackIds(GetFullTracks(tracksBefore))
                update := NewPlaylistUpdate(idsBefore)

                for _, id := range idsAfter {
                    update.AddTrack(id)
                }

				idsToAddSlice := update.GetIdsToAdd()
				idsToRemoveSlice := update.GetIdsToRemove()

				for i := 0; i < len(idsToRemoveSlice); i += 50 {
					batch := idsToRemoveSlice[i:min(i+50, len(idsToRemoveSlice))]
					spotifyClient.RemoveTracksFromPlaylist("jlewalle", playlist.ID, batch...)
				}

				for i := 0; i < len(idsToAddSlice); i += 50 {
					batch := idsToAddSlice[i:min(i+50, len(idsToAddSlice))]
					spotifyClient.AddTracksToPlaylist("jlewalle", playlist.ID, batch...)
				}
			}
		}

		if !options.GuessOnly {
			SendEmail(buffer.String())
		}
	}

	e := NewEclectic24()
    err = UpdateEclectic(spotifyClient, e)
    if err != nil {
        log.Fatalf("%v\n", err)
    }
}
