package main

import (
	"log"
	"encoding/json"
	"io/ioutil"
)

type Region struct {
	Id string
	Region string
	VenueIds []string
}

func LoadRegions() (regions []Region) {
	log.Printf("Reading regions.json...")

	file, e := ioutil.ReadFile("./regions.json")
	if e != nil {
		log.Fatal("File error: %v\n", e)
	}

	json.Unmarshal(file, &regions)

	return
}
