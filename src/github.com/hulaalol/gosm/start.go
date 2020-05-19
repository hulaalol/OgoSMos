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

	// TEST AREA

	//var item = "Viscount_Morpeth"
	//var q = genItem(item)

	//q = genItem("Waterloo_Bridge")

	//q = genItem("Holyoak")

	//q = genItem("Dugard")

	//item = "Morpeth_Dock"
	//q = genItem(item)

	//item = "Morpeth"
	//q = genItem(item)

	//var item = "Lambeth_Bridge"
	//var q = genItem(item)

	//var q = genItem("Holyoak", true) //Stanford_University")
	//var q = genItem("Lambeth_Bridge", true)
	//fmt.Println(q)

	//var q = genQuestion("Lloyd")
	//fmt.Println(q)

	/*
		item = "River_Thames"
		q = genItem(item)

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
	*/
	// select subject matching class of item

	//var q = genQuestion("Bel,_Syria")
	//fmt.Println(q)

	//var q = genQuestion("Duke")
	//fmt.Println(q)

	//q = genQuestion("Queen_Victoria")
	//fmt.Println(q)

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
