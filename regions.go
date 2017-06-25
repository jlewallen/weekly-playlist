package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type Region struct {
	Id       string
	Region   string
	VenueIds []string
}

func LoadRegions(fileName string) (regions []Region) {
	log.Printf("Reading %s...", fileName)

	file, e := ioutil.ReadFile(fileName)
	if e != nil {
		log.Fatal("File error: %v\n", e)
	}

	json.Unmarshal(file, &regions)

	return
}
