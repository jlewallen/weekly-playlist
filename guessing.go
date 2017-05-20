package main

import (
	"regexp"
	"strings"
)

type ArtistGuess struct {
	Name string
	Children []*ArtistGuess
}

func FlattenArtist(root *ArtistGuess) (artists []*ArtistGuess) {
	artists = append(artists, root.Children...)

	for _, child := range root.Children {
		artists = append(artists, FlattenArtist(child)...)
	}

	return
}

func SwapAndsPermutation(guess *ArtistGuess) {
	andSwapped := regexp.MustCompile("&").ReplaceAllString(guess.Name, "and")
	if andSwapped != guess.Name {
		newGuess := ArtistGuess{ Name: strings.TrimSpace(andSwapped) }
		if len(newGuess.Name) > 0 {
			guess.Children = append(guess.Children, &newGuess)
		}
	}
}

func SeparateMultipleArtistsPermutation(guess *ArtistGuess) {
	moreNames := regexp.MustCompile("(?i)(?:\\\\w|w\\/|,\\s+|\\bAND\\b|\\bY\\b|&|:)").Split(guess.Name, -1)
	if len(moreNames) > 1 {
		for _, name := range moreNames {
			name = strings.TrimSpace(name)
			if len(name) > 0 {
				guess.Children = append(guess.Children, &ArtistGuess{ Name: name })
			}
		}
	}
}

func InitialPermutation(guess *ArtistGuess) {
	splitRe := regexp.MustCompile("(?i)(?:\\b\\\\w\\b|\\bW/\\b|,\\s+|\\b\\+ MORE\\b|\\sWITH\\sSPECIAL\\sGUEST\\s|\\bWITH\\b|\\bFEAT\\b|\\s+FT\\.?\\s+|\\bPRESENTS?\\b|\\bFEATURING\\b|\\||\\/\\/?)")
	split := splitRe.Split(guess.Name, -1)
	if len(split) > 1 {
		for _, substring := range split {
			child := ArtistGuess{Name: strings.TrimSpace(substring)}
			if len(child.Name) > 0 {
				SwapAndsPermutation(&child)
				SeparateMultipleArtistsPermutation(&child)
				guess.Children = append(guess.Children, &child)
			}
		}
	}
}

func RemoveVenuePermutation(guess *ArtistGuess) {
	anotherName := regexp.MustCompile("(?i)(\\s+AT\\s+.+)").ReplaceAllString(guess.Name, "")
	if anotherName != guess.Name {
		newGuess := ArtistGuess{ Name: anotherName }
		InitialPermutation(&newGuess)
		guess.Children = append(guess.Children, &newGuess)
	}
}

func GuessArtistsForEvent(title string) (guess *ArtistGuess) {
	patternsToRemove := []string{
		"\\bLOS ANGELES, CA\\b",
		"\\bNEW HAVEN, CT\\b",
		"\\b(SOLD\\s+OUT\\s*!+)",
		"\\b(SOLD\\s+OUT:)",
		"^(SOLD\\s+OUT)",
		"\\bLIVE (AT|@).+",
		"\\|.+",
		"\\(.+\\)",
		"�",
	}

	guess = &ArtistGuess{Name: title}

	for _, pattern := range patternsToRemove {
		r := regexp.MustCompile("(?i)" + pattern)
		title = r.ReplaceAllString(title, "")
	}

	title = strings.TrimSpace(title)
	cleaned := ArtistGuess{Name: title}
	InitialPermutation(&cleaned)
	RemoveVenuePermutation(&cleaned)
	guess.Children = append(guess.Children, &cleaned)

	return guess
}
