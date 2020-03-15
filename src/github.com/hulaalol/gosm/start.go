package main

import (
	"log"
	"time"
)

func main() {

	file := "data/london.osm.pbf"

	//defer profile.Start().Stop()

	//fmt.Println(testGet())
	//fmt.Println(cleanStreetname("Hauptstätter Straße"))
	queryDBP("Walnut")

	startTime := time.Now()
	log.Printf("start parsing")
	decoder(file)

	log.Printf("building edges in")

	var edgeErrors = 0

	elapsed := time.Since(startTime)
	log.Printf("\rfinished parsing graph and building edgesIn in %s", elapsed)
	log.Printf("converted strange edges to points: %d", unusualEdges)
	log.Printf("no edges in for: %d", edgeErrors)

	startWebserver()

}
