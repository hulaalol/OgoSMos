package main

import (
	"fmt"
	"log"
	"time"
)

func main() {

	file := "data/london.osm.pbf"

	//defer profile.Start().Stop()

	//fmt.Println(testGet())
	//fmt.Println(cleanStreetname("Hauptstätter Straße"))

	// TEST AREA

	var item = "Lambeth_Bridge"
	var q = genItem(item)

	item = "Walnut"
	q = genItem(item)

	item = "McLaren"
	q = genItem(item)

	item = "Lambeth"
	q = genItem(item)

	item = "Pall_Mall,_London"
	q = genItem(item)

	item = "Morpeth_School"
	q = genItem(item)
	fmt.Println(q)

	// select subject matching class of item

	// END TEST AREA

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
