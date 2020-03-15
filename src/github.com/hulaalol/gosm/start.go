package main

import (
	"log"
	"time"
)

func main() {
	//defer profile.Start().Stop()
	startTime := time.Now()
	log.Printf("start parsing")
	decoder()

	log.Printf("building edges in")

	var edgeErrors = 0

	elapsed := time.Since(startTime)
	log.Printf("\rfinished parsing graph and building edgesIn in %s", elapsed)
	log.Printf("converted strange edges to points: %d", unusualEdges)
	log.Printf("no edges in for: %d", edgeErrors)

	startWebserver()

}
