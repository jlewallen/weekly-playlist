package main

import (
	mapset "github.com/deckarep/golang-set"
	fb "github.com/huandu/facebook"
	"fmt"
	"github.com/zmb3/spotify"
	"log"
	"os"
	"strings"
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

func (resolver *ArtistResolver) GetSpotifyArtistsForGuess(depth int, artist *ArtistGuess) {
	log.Printf("      |%s%s\n", strings.Repeat("  ", depth), artist.Name)

	if resolver.artistCache[artist.Name] == nil {
		country := "US"
		found, err := spotify.SearchOpt(artist.Name, spotify.SearchTypeArtist, &spotify.Options{Country: &country})
		if err != nil {
			log.Printf("Error:", err)
		} else {
			if found.Artists != nil {
				for _, item := range found.Artists.Artists {
					if strings.ToLower(item.Name) == strings.ToLower(artist.Name) {
						log.Printf("      #%s%s\n", strings.Repeat("  ", depth), item.Name)
						resolver.artistCache[artist.Name] = &item
						resolver.spotifyArtists[artist.Name] = &item
						break
					}
				}
			}
		}
	} else {
		resolver.spotifyArtists[artist.Name] = resolver.artistCache[artist.Name]
	}

	for _, child := range artist.Children {
		resolver.GetSpotifyArtistsForGuess(depth+1, child)
	}
}

func (resolver *ArtistResolver) GetSpotifyArtists(event Event) (spotifyArtists map[string]*spotify.FullArtist) {
	resolver.spotifyArtists = make(map[string]*spotify.FullArtist)

	resolver.GetSpotifyArtistsForGuess(0, event.Artists)

	spotifyArtists = resolver.spotifyArtists

	return
}

func (resolver *ArtistResolver) GetArtistTracks(artist spotify.FullArtist) (tracks []spotify.FullTrack) {
	topTracks, _ := spotify.GetArtistsTopTracks(artist.ID, "US")
	if len(topTracks) > 3 {
		tracks = topTracks[:3]
	} else {
		tracks = topTracks
	}
	return
}

func GetPlaylistByTitle(spotifyClient *spotify.Client, name string) (playlist spotify.SimplePlaylist, err error) {
	playlists, err := spotifyClient.GetPlaylistsForUser("jlewalle")
	if err == nil {
		if playlists.Playlists != nil {
			for _, iter := range playlists.Playlists {
				if iter.Name == name {
					playlist = iter
				}
			}
		}
	}

	return
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

func main() {
	f, err := os.OpenFile("weekly.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(os.Stdout)
	// log.SetOutput(f)

	spotifyClient, _ := AuthenticateSpotify()
	facebookSession, _ := AuthenticateFacebook()
	regions := LoadRegions()

	tracksAfter := []spotify.FullTrack{}

	artistsResolver := NewArtistResolver()

	for _, region := range regions {
		for _, venueId := range region.VenueIds {
			for _, event := range ProcessVenue(facebookSession, venueId) {
				log.Printf("   '%s'\n", event.Name)
				artists := artistsResolver.GetSpotifyArtists(event)
				for _, value := range artists {
					artistTracks := artistsResolver.GetArtistTracks(*value)
					tracksAfter = append(tracksAfter, artistTracks...)
					log.Printf("   %d tracks '%s''\n", len(artistTracks), value.Name)
				}
			}
		}

		idsAfter := GetTrackIds(tracksAfter)

		playlist, err := GetPlaylistByTitle(spotifyClient, region.Region+" weekly")
		if err != nil {
			log.Printf("%v\n", err)
			os.Exit(1)
		}
		tracksBefore, _ := GetPlaylistTracks(spotifyClient, "jlewalle", playlist.ID)
		idsBefore := GetTrackIds(GetFullTracks(tracksBefore))

		beforeSet := mapset.NewSetFromSlice(MapIds(idsBefore))
		afterSet := mapset.NewSetFromSlice(MapIds(idsAfter))

		idsToAdd := afterSet.Difference(beforeSet)
		idsToRemove := beforeSet.Difference(afterSet)

		idsToAddSlice := idsToAdd.ToSlice()
		idsToRemoveSlice := idsToRemove.ToSlice()

		log.Printf("%s before=%d after=%d adding=%d removing=%d\n",
			playlist.Name,
			len(tracksBefore),
			len(tracksAfter),
			len(idsToAddSlice),
			len(idsToRemoveSlice))

		for i := 0; i < len(idsToRemoveSlice); i += 50 {
			batch := idsToRemoveSlice[i:min(i+50, len(idsToRemoveSlice))]
			fmt.Printf("removing %v\n", len(batch))
			spotifyClient.RemoveTracksFromPlaylist("jlewalle", playlist.ID, ToSpotifyIds(batch)...)
		}

		for i := 0; i < len(idsToAddSlice); i += 50 {
			batch := idsToAddSlice[i:min(i+50, len(idsToAddSlice))]
			fmt.Printf("adding %v\n", len(batch))
			spotifyClient.AddTracksToPlaylist("jlewalle", playlist.ID, ToSpotifyIds(batch)...)
		}

	}
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
