package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

const redirectURI = "http://localhost:8080/callback"

var (
	auth  = spotifyauth.New(spotifyauth.WithRedirectURL(redirectURI), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate))
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

func main() {
	// Get user authorization

	// first start an HTTP server
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	// wait for auth to complete
	client := <-ch

	// use the client to make calls that require authorization
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)

	playlistID := os.Getenv("SPOTIFY_PLAYLIST_ID")

	
	// Get the first 100 tracks in the playlist
	tracks, err := client.GetPlaylistTracks(playlistID)
	if err != nil {
		log.Fatalf("error retrieve playlist tracks: %v", err)
	}

	// Print the tracks in current order
	fmt.Println("Original tracks:")
	for _, track := range tracks.Tracks {
		fmt.Println( "- ", track.Track.ID)
	}

	// Create a new slice of tracks ID to reorder
	var newTracks []spotify.ID
	for _, track := range tracks.Tracks {
		newTracks = append(newTracks, track.Track.ID)
	}

	// Shuffle the tracks
	for i := range newTracks {
		j := rand.Intn(i + 1)
		newTracks[i], newTracks[j] = newTracks[j], newTracks[i]
	}

	// Print the new track IDs
	fmt.Println("New track IDs:")
	for _, trackID := range newTracks {
		fmt.Println( "- ", trackID)
	}

	// Replace the tracks in the playlist with the new order
	err = client.ReplacePlaylistTracks(playlistID, newTracks...)
	if err != nil {
		log.Fatalf("error replace playlist tracks: %v", err)
	}
	
	fmt.Println("Playlist shuffled!")
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}

	// use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok))
	fmt.Fprintf(w, "Login Completed!")
	ch <- client
}