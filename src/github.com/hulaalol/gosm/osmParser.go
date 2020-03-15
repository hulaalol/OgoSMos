package main

import (
	"log"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"

	"github.com/thomersch/gosmparse"
)

// constants
const nofNodes = 50000000
const nofNodes2 = 7000000
const nofEdges = 8000000

var finalNodes []Node
var finalEdgesOut []Edge
var finalOffsetsOut []uint32
var unusualEdges int

var start Node
var finish Node

// geodistance functions
func QuickDistance(lat float32, lng float32, lat0 float32, lng0 float32) float64 {
	deglen := 110.25
	x := float64(lat - lat0)
	y := float64((lng - lng0)) * math.Cos(float64(lat0))
	return (deglen * math.Sqrt(x*x+y*y) / 1000)
}

func geoDistancePrecision(lat1 float32, lng1 float32, lat2 float32, lng2 float32) float64 {
	const PI float64 = 3.141592653589793

	radlat1 := float64(PI * float64(lat1) / 180)
	radlat2 := float64(PI * float64(lat2) / 180)

	theta := float64(lng1 - lng2)
	radtheta := float64(PI * theta / 180)

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	if dist < 0 {
		dist = 0
	}
	dist = math.Acos(dist)
	dist = dist * 180 / PI
	dist = dist * 60 * 1.1515
	dist = dist * 1.609344 * 1000
	return dist
}

func geoDistance(lat1 float32, lng1 float32, lat2 float32, lng2 float32) uint16 {
	const PI float64 = 3.141592653589793

	radlat1 := float64(PI * float64(lat1) / 180)
	radlat2 := float64(PI * float64(lat2) / 180)
	theta := float64(lng1 - lng2)
	radtheta := float64(PI * theta / 180)
	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	if dist < 0 {
		dist = 0
	}

	dist = math.Acos(dist)
	dist = dist * 180 / PI
	dist = dist * 60 * 1.1515
	dist = dist * 1.609344 * 1000

	result := uint16(math.Round(dist))
	return result
}

// Implement the gosmparser.OSMReader interface here.
// Streaming data will call those functions.
type Empty struct {
}

type Node struct {
	id  int64
	idx uint32
	lat float32
	lon float32
}

type Edge struct {
	n1         int64
	n2         int64
	n2idx      uint32
	distance   uint16
	speed      uint8
	access     uint8
	streetname string
}

type dataHandlerNodes struct {
	nodeCount int
	nodes     *[nofNodes2]Node
	nodesDict *map[int64]Empty
	mutex     *sync.Mutex
}

type dataHandlerRelations struct {
}

type dataHandlerWays struct {
	nodeCount    int
	wayCount     int
	nodes        *[nofNodes]int64
	edgesOut     *[nofEdges]Edge
	mutex        *sync.Mutex
	accessRights *map[string]map[string]Empty
}

// data handler for nodes
func (d *dataHandlerNodes) ReadNode(n gosmparse.Node) {
	if _, ok1 := (*d.nodesDict)[n.Element.ID]; ok1 {
		d.mutex.Lock()
		d.nodes[d.nodeCount] = Node{n.Element.ID, 0, float32(n.Lat), float32(n.Lon)}
		d.nodeCount++
		d.mutex.Unlock()
	}
}
func (d *dataHandlerNodes) ReadWay(w gosmparse.Way) {
}
func (d *dataHandlerNodes) ReadRelation(r gosmparse.Relation) {
}

// data handler for ways
func (d *dataHandlerWays) ReadNode(n gosmparse.Node) {
}

func (d *dataHandlerWays) ReadWay(w gosmparse.Way) {

	var m int = 50
	var name string = "UNDEFINED"
	if _, ok1 := w.Element.Tags["highway"]; ok1 {

		if _, ok3 := (*d.accessRights)["blacklist"][w.Element.Tags["highway"]]; ok3 {
			return
		}
		// parse maxspeed
		if v2, ok2 := w.Element.Tags["maxspeed"]; ok2 {
			ms, err := strconv.Atoi(v2)
			if err != nil {
				ms = 50
			}
			if ms < 0 {
				ms = 50
			}
			m = ms
		}

		access := 0
		//parse access right
		if _, ok := (*d.accessRights)["carOnly"][w.Element.Tags["highway"]]; ok {
			access = 1
		}
		if _, ok := (*d.accessRights)["bikeOnly"][w.Element.Tags["highway"]]; ok {
			access = 2
		}
		if _, ok := (*d.accessRights)["footbike"][w.Element.Tags["highway"]]; ok {
			access = 3
		}
		if _, ok := (*d.accessRights)["footOnly"][w.Element.Tags["highway"]]; ok {
			access = 4
		}
		d.mutex.Lock()
		for i := 0; i < len(w.NodeIDs)-1; i++ {

			if v2, ok2 := w.Element.Tags["name"]; ok2 {
				name = v2
			}

			d.edgesOut[d.wayCount] = Edge{w.NodeIDs[i], w.NodeIDs[i+1], 0, 0, uint8(m), uint8(access), name}
			d.wayCount++

			//edges in
			d.edgesOut[d.wayCount] = Edge{w.NodeIDs[i+1], w.NodeIDs[i], 0, 0, uint8(m), uint8(access), name}
			d.wayCount++

			d.nodes[d.nodeCount] = w.NodeIDs[i]
			d.nodeCount++
			d.nodes[d.nodeCount] = w.NodeIDs[i+1]
			d.nodeCount++

		}
		d.mutex.Unlock()

	}

}
func (d *dataHandlerWays) ReadRelation(r gosmparse.Relation) {

}

func decoder(m string) {
	//start := time.Now()

	//file := "data/stgt.osm.pbf"
	file := m

	log.Printf("parsing edges ...")
	r, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	dec := gosmparse.NewDecoder(r)
	// Parse will block until it is done or an error occurs.

	var nodes [nofNodes]int64
	var edgesOut [nofEdges]Edge

	// blacklist
	blacklist := make(map[string]Empty)
	b := [9]string{"bridleway", "escape", "raceway", "service", "proposed", "construction", "elevator", "track", "platform"}
	for i := 0; i < len(b); i++ {
		blacklist[b[i]] = Empty{}
	}
	carOnly := make(map[string]Empty)
	c := [8]string{"primary", "trunk", "motorway", "motorway_link", "trunk_link", "primary_link"}
	for i := 0; i < len(c); i++ {
		carOnly[c[i]] = Empty{}
	}
	bikeOnly := make(map[string]Empty)
	bikeOnly["cycleway"] = Empty{}

	footbike := make(map[string]Empty)
	footbike["path"] = Empty{}

	footOnly := make(map[string]Empty)
	f := [3]string{"footway", "steps", "pedestrian"}
	for i := 0; i < len(f); i++ {
		footOnly[f[i]] = Empty{}
	}

	accessRights := make(map[string]map[string]Empty)
	accessRights["blacklist"] = blacklist
	accessRights["carOnly"] = carOnly
	accessRights["bikeOnly"] = bikeOnly
	accessRights["footOnly"] = footOnly
	accessRights["footbike"] = footbike

	d := dataHandlerWays{0, 0, &nodes, &edgesOut, &sync.Mutex{}, &accessRights}
	err = dec.Parse(&d)

	log.Printf("building nodesDict ...")
	var nodesDict = make(map[int64]Empty)
	for i := 0; i < len(nodes)-1; i++ {
		nodesDict[nodes[i]] = Empty{}
	}

	log.Printf("get nodes location")
	r, err = os.Open(file)

	var nodes2 [nofNodes2]Node

	d2 := dataHandlerNodes{0, &nodes2, &nodesDict, &sync.Mutex{}}
	dec = gosmparse.NewDecoder(r)
	err = dec.Parse(&d2)

	nodesDict = nil
	runtime.GC()

	var idx = 0
	for i := len(nodes2) - 1; i >= 0; i-- {
		if nodes2[i].id > 0 {
			idx = i
			log.Printf("last node %d", idx)
			break
		}
	}

	nodes3 := nodes2[0:idx]

	idx = 0
	for i := len(edgesOut) - 1; i >= 0; i-- {
		if edgesOut[i].n1 > 0 {
			idx = i
			log.Printf("last edge %d", idx)
			break
		}
	}

	edgesOut2 := edgesOut[0:idx]
	log.Printf("done ...")

	log.Printf("sorting nodes")
	sort.Slice(nodes3[:], func(i, j int) bool {
		return nodes3[i].id < nodes3[j].id
	})

	log.Printf("sorting edges by source")
	sort.Slice(edgesOut2[:], func(i, j int) bool {
		return edgesOut2[i].n1 < edgesOut2[j].n1
	})

	log.Printf("calc offsets for outgoing edges")
	offsets := make([]uint32, 0)
	offsets = append(offsets, 0)

	for i := 0; i <= len(nodes3)-1; i++ {

		if nodes3[i].id < edgesOut2[offsets[i]].n1 {
			offsets = append(offsets, offsets[i])
			continue
		}
		var currOffset = offsets[i]
		for nodes3[i].id == edgesOut2[currOffset].n1 {
			var c1Lon = nodes3[i].lon
			var c1Lat = nodes3[i].lat
			// search in sorted array nodes3.id the value of edgesOut2[currOffset].n2
			var c2I = sort.Search(len(nodes3)-1, func(k int) bool { return (int(edgesOut2[currOffset].n2) <= int(nodes3[k].id)) })

			if edgesOut2[currOffset].n2 == nodes3[c2I].id {
				var c2Lon = nodes3[c2I].lon
				var c2Lat = nodes3[c2I].lat

				var d = geoDistance(c1Lat, c1Lon, c2Lat, c2Lon)

				edgesOut2[currOffset].distance = d
			}
			currOffset = currOffset + 1

			if currOffset == uint32(len(edgesOut2)) {
				break
			}
		}
		offsets = append(offsets, currOffset)
	}

	finalNodes = nodes3

	for index, _ := range finalNodes {
		finalNodes[index].idx = uint32(index)
	}

	finalEdgesOut = edgesOut2

	for index, _ := range finalEdgesOut {
		finalEdgesOut[index].n2idx = uint32(sort.Search(len(nodes3)-1, func(k int) bool { return finalEdgesOut[index].n2 <= nodes3[k].id }))
	}

	finalOffsetsOut = offsets[0 : len(finalNodes)+1]

	debug := false
	if debug {
		n := 10
		log.Printf("%d", len(finalNodes))
		log.Printf("%d", len(finalEdgesOut))
		log.Printf("%d", len(finalOffsetsOut))

		log.Printf("%v", nodes3[0:n])
		log.Printf("%v", edgesOut2[0:n])
		log.Printf("%v", offsets[0:n])
	}

	if err != nil {
		panic(err)
	}

}
