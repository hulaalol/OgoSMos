package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/profile"
	"github.com/thomersch/gosmparse"

	hp "container/heap"
)

// constants
const nofNodes = 500000000
const nofNodes2 = 70000000
const nofEdges = 80000000

var finalNodes []Node
var finalEdgesOut []Edge
var finalOffsetsOut []uint32

//var finalEdgesIn map[uint32]*EdgesIn
//var finalEdgesIn []Edge

//var finalEdgesIn map[uint32][]EdgeIn

var unusualEdges int

var start Node
var finish Node

// geodistance function

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

type ParseNode struct {
	id  int64
	lat float32
	lon float32
}

type ParseEdge struct {
	n1 int64
	n2 int64
}

type Node struct {
	id  int64
	idx uint32
	lat float32
	lon float32
}

type Edge struct {
	n1       int64
	n2       int64
	n2idx    uint32
	distance uint16
	speed    uint8
	access   uint8
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
		//delete((*d.nodesDict), n.Element.ID)
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

			d.edgesOut[d.wayCount] = Edge{w.NodeIDs[i], w.NodeIDs[i+1], 0, 0, uint8(m), uint8(access)}
			d.wayCount++

			//edges in
			d.edgesOut[d.wayCount] = Edge{w.NodeIDs[i+1], w.NodeIDs[i], 0, 0, uint8(m), uint8(access)}
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

func decoder() {
	//start := time.Now()

	//file := "data/stgt.osm.pbf"
	file := "data/germany.osm.pbf"

	fmt.Println("parsing edges ...")
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
	//b := [6]string{"track", "bus_guideway", "bridleway", "escape", "raceway", "steps"}
	b := [9]string{"bridleway", "escape", "raceway", "service", "proposed", "construction", "elevator", "track", "platform"}
	for i := 0; i < len(b); i++ {
		blacklist[b[i]] = Empty{}
	}

	carOnly := make(map[string]Empty)
	c := [6]string{"primary", "trunk", "motorway", "motorway_link", "trunk_link", "primary_link"}
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

	//err = dec.Parse(&dataHandler{})
	err = dec.Parse(&d)

	fmt.Println("building nodesDict ...")
	var nodesDict = make(map[int64]Empty)
	for i := 0; i < len(nodes)-1; i++ {
		nodesDict[nodes[i]] = Empty{}
	}

	//debug.FreeOSMemory()
	//runtime.GC()

	fmt.Println("get nodes location")
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
	//nodes3 := make([]Node, idx)
	//copy(nodes3, nodes2[0:idx])

	idx = 0
	for i := len(edgesOut) - 1; i >= 0; i-- {
		if edgesOut[i].n1 > 0 {
			idx = i
			log.Printf("last edge %d", idx)
			break
		}
	}

	edgesOut2 := edgesOut[0:idx]
	//edgesOut2 := make([]Edge, idx)
	//copy(edgesOut2, edgesOut[0:idx])

	fmt.Println("done ...")

	fmt.Println("sorting nodes")
	sort.Slice(nodes3[:], func(i, j int) bool {
		return nodes3[i].id < nodes3[j].id
	})

	fmt.Println("sorting edges by source")
	sort.Slice(edgesOut2[:], func(i, j int) bool {
		return edgesOut2[i].n1 < edgesOut2[j].n1
	})

	fmt.Println("calc offsets for outgoing edges")
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

				if d > 2000 {
					fmt.Println("warn: strange long edge, converting to point")
					unusualEdges++
					edgesOut2[currOffset].n2 = edgesOut2[currOffset].n1
				}

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

	//elapsed := time.Since(start)
	//log.Printf("\rfinished parsing and building graph in %s", elapsed)

	debug := true
	if debug {
		n := 10
		//for index, element := range nodes3 {
		//	if index == n {
		//		break
		//	}
		//	fmt.Println(edgesIn[element.id])
		//}

		fmt.Println(len(finalNodes))
		fmt.Println(len(finalEdgesOut))
		fmt.Println(len(finalOffsetsOut))

		fmt.Printf("%v", nodes3[0:n])
		fmt.Println("\r")
		fmt.Printf("%v", edgesOut2[0:n])
		fmt.Println("\r")
		fmt.Printf("%v", offsets[0:n])

	}

	if err != nil {
		panic(err)
	}

}

type LeafletEdge struct {
	C []float32
}

type LeafletEdgeArray struct {
	Name string
	Data []LeafletEdge
}

type LeafletEdgeArrayDijkstra struct {
	Name     string
	Data     []LeafletEdge
	Distance uint32
}

func filterEdges(NWtlLat float32, NWtlLon float32, SEbrLat float32, SEbrLon float32, zoomLevel uint16) []LeafletEdge {
	fmt.Println("start filtering edges")

	fmt.Println(len(finalEdgesOut))
	fmt.Println(len(finalOffsetsOut))
	fmt.Println(len(finalNodes))

	var e = make([]LeafletEdge, 0)

	for index, element := range finalNodes {
		if element.lat < NWtlLat && element.lat > SEbrLat && element.lon > NWtlLon && element.lon < SEbrLon {

			var e0 = finalEdgesOut[finalOffsetsOut[index]:finalOffsetsOut[index+1]]
			//fmt.Println(e0)

			for _, j := range e0 {
				//find locs

				//only carOnly nodes in big zoom
				if zoomLevel < 14 {
					if j.access != 1 {
						continue
					}
				}
				var i1 = sort.Search(len(finalNodes)-1, func(k int) bool { return j.n1 <= finalNodes[k].id })
				var i2 = sort.Search(len(finalNodes)-1, func(k int) bool { return j.n2 <= finalNodes[k].id })

				e = append(e, LeafletEdge{[]float32{finalNodes[i1].lat, finalNodes[i1].lon, finalNodes[i2].lat, finalNodes[i2].lon}})
				//fmt.Println("appended edge")
			}

		}

	}
	fmt.Println("this many edges are visible on the map:")
	fmt.Println(len(e))
	return e

}

func convLeafletEdge2JSONDijkstra(data []LeafletEdge, distance uint32) []byte {

	fmt.Println("len of data to convert:")
	fmt.Println(len(data))

	profile := LeafletEdgeArrayDijkstra{"edges", data, distance}
	js, err := json.Marshal(profile)
	if err != nil {
		fmt.Println("error while converting LeafletEdgeArray to JSON")
		return []byte{0}
	}
	return js
}

func convLeafletEdge2JSON(data []LeafletEdge) []byte {

	fmt.Println("len of data to convert:")
	fmt.Println(len(data))

	profile := LeafletEdgeArray{"edges", data}
	js, err := json.Marshal(profile)
	if err != nil {
		fmt.Println("error while converting LeafletEdgeArray to JSON")
		return []byte{0}
	}
	return js
}

// dijkstra

// heap implementation

type HeapNode struct {
	idx uint32
	lat float32
	lon float32
}

//type path struct {
//	value     uint32
//	heuristic float32
//	nodes     []HeapNode
//nodes     []Node
//}

type path struct {
	value     uint32
	heuristic float64
	node      uint32
}

type minPath []path

func (h minPath) Len() int           { return len(h) }
func (h minPath) Less(i, j int) bool { return h[i].heuristic < h[j].heuristic }
func (h minPath) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *minPath) Push(x interface{}) {
	*h = append(*h, x.(path))
}

func (h *minPath) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type heap struct {
	values *minPath
}

func newHeap() *heap {
	return &heap{values: &minPath{}}
}

func (h *heap) push(p path) {
	hp.Push(h.values, p)
}

func (h *heap) pop() path {
	i := hp.Pop(h.values)
	return i.(path)
}

func findClosestNode(lat float32, lon float32, delta uint16) int {
	var minIdx = 0
	var minD = 999999999.0

	var cDelta = float32(0.00001)

	for idx, node := range finalNodes {

		if node.lat-lat > cDelta || node.lon-lon > cDelta {
			continue
		}

		var d = geoDistancePrecision(lat, lon, node.lat, node.lon)
		//fmt.Println(d)

		if d < float64(delta) {
			return idx
		}

		if d < float64(minD) {
			minD = d
			minIdx = idx
		}

	}
	return minIdx
}

type tracker struct {
	came_from   uint32
	cost_so_far uint32
}

func getPath(origin Node, destiny Node) (uint32, []Node) {

	//startTime := time.Now()

	originHPN := HeapNode{origin.idx, origin.lat, origin.lon}
	destinyHPN := HeapNode{destiny.idx, destiny.lat, destiny.lon}

	h := newHeap()

	//var upperBound = float32(geoDistancePrecision(destiny.lat, destiny.lon, origin.lat, origin.lon))
	//h.push(path{value: 0, heuristic: upperBound, nodes: []HeapNode{HeapNode{origin.idx, origin.lat, origin.lon}}})

	//var infValue float32 = 99999999

	//start := HeapNode{, origin.lat, origin.lon}
	h.push(path{value: 0, heuristic: 0, node: origin.idx})

	//came_from := make(map[uint32]uint32)
	//cost_so_far := make(map[uint32]uint32)

	t := make(map[uint32]tracker)

	t[start.idx] = tracker{0, 0}
	//came_from[start.idx] = start.idx
	//cost_so_far[start.idx] = 0

	//visited := make(map[uint32]bool)

	for len(*h.values) > 0 {

		//if time.Since(startTime).Seconds() > 20.0 {
		//	fmt.Println("timeout, please try again...")
		//	return 0, make([]Node, 0)
		//}

		// Find the nearest yet to visit node
		p := h.pop()
		//fmt.Println("path prio", p.heuristic)
		node := p.node

		if node == destinyHPN.idx {
			// return, we found the path

			c := destinyHPN.idx

			path := make([]uint32, 0)

			for c != originHPN.idx {
				path = append(path, c)
				c = t[c].came_from
			}

			var result = make([]Node, len(path))
			for idx, n := range path {
				result[idx] = Node{0, n, finalNodes[n].lat, finalNodes[n].lon}
			}
			//fmt.Println("gesamtstrecke: s", upperBound)
			return p.value, result
			//return p.value, p.nodes
		}

		// collect edges
		var eOutIdxStart = finalOffsetsOut[node]
		var eOutIdxEnd = finalOffsetsOut[node+1]
		var eOut = finalEdgesOut[eOutIdxStart:eOutIdxEnd]

		//var eOutF = make([]Edge, 1)
		//
		//for _, e := range eOut {
		//
		//	if e.access == 0 || e.access == 1 {
		//		eOutF = append(eOutF, e)
		//	}
		//
		//}

		//if _, ok := finalEdgesIn[node.idx]; ok {
		//	var eInContainer = finalEdgesIn[node.idx].eIn

		//	for _, eI := range eInContainer {
		//		//eOut = append(eOut, Edge{finalNodes[eI.n1idx].id, finalNodes[node.idx].id, node.idx, eI.distance, eI.speed, eI.access})
		//		eOut = append(eOut, Edge{finalNodes[node.idx].id, finalNodes[eI.n1idx].id, eI.n1idx, eI.distance, eI.speed, eI.access})

		//	}
		//

		//fmt.Println(eOut)
		for _, e := range eOut {

			new_cost := t[node].cost_so_far + uint32(e.distance)

			exists := false

			if _, ok := t[e.n2idx]; ok {
				exists = true
			}

			if !exists || (new_cost < t[e.n2idx].cost_so_far) {
				//if new_cost < cost_so_far[e.n2idx] {

				cost_so_far := new_cost
				//heureka := float32(geoDistancePrecision(destiny.lat, destiny.lon, finalNodes[e.n2idx].lat, finalNodes[e.n2idx].lon))

				heureka := math.Pow(QuickDistance(destiny.lat, destiny.lon, finalNodes[e.n2idx].lat, finalNodes[e.n2idx].lon), 4)
				//fmt.Println("luftlinie zum ziel: ", heureka)
				prio := float64(new_cost) + heureka

				//hpn := HeapNode{finalNodes[e.n2idx].idx, finalNodes[e.n2idx].lat, finalNodes[e.n2idx].lon}
				h.push(path{value: new_cost, heuristic: prio, node: finalNodes[e.n2idx].idx})
				came_from := p.node

				t[e.n2idx] = tracker{came_from, cost_so_far}
			}

		}
	}

	fmt.Println("can't find a way :(")
	return 0, make([]Node, 0)
}

// web server declaration
type Profile struct {
	Name    string
	Hobbies []string
}

type EdgesJSON struct {
	Name  string
	Edges [][]int64
}

type Command struct {
	Do        string  `json:"do"`
	ZoomLevel uint16  `json:"zoomLevel"`
	NWtlLat   float32 `json:"NWtlLat"`
	NWtlLon   float32 `json:"NWtlLon"`
	SWblLat   float32 `json:"SWblLat"`
	SWblLon   float32 `json:"SWblLon"`

	// these are not really needed (backup)
	SEbrLat float32 `json:"SEbrLat"`
	SEbrLon float32 `json:"SEbrLon"`
	NEtrLat float32 `json:"NEtrLat"`
	NEtrLon float32 `json:"NEtrLon"`
}

type DijkstraInput struct {
	Do        string  `json:"do"`
	Mode      string  `json:"mode"`
	StartLat  float32 `json:"startLat"`
	StartLon  float32 `json:"startLon"`
	TargetLat float32 `json:"targetLat"`
	TargetLon float32 `json:"targetLon"`
}

func dijkstra(w http.ResponseWriter, r *http.Request) {
	log.Println("received dijkstra request")

	switch r.Method {
	case "GET":
		log.Println("there is no GET method for dijkstra!")
	case "POST":
		//log.Println(r.Body)
		decoder := json.NewDecoder(r.Body)
		var req DijkstraInput
		err := decoder.Decode(&req)
		if err != nil {
			panic(err)
		}
		//log.Println(req.Do)
		//log.Println(req.Mode)

		//build answer here
		//var startNode = finalNodes[findClosestNode(req.StartLat, req.StartLon, 10)]
		//log.Println(startNode)
		//var destNode = finalNodes[findClosestNode(req.TargetLat, req.TargetLon, 10)]
		//log.Println(destNode)

		startTime := time.Now()
		//var distance, path = getPath(startNode, destNode)
		var distance, path = getPath(start, finish)

		log.Printf("found path with %d nodes and %d m length", len(path), distance)

		elapsed := time.Since(startTime)
		log.Printf("\rfinished finding dijkstra way in %s", elapsed)

		var coords = make([]LeafletEdge, 0)

		for idx, _ := range path {

			if idx != len(path)-1 {
				//var nodeIdx1 = sort.Search(len(finalNodes)-1, func(k int) bool { return path[idx] <= finalNodes[k].id })
				//var nodeIdx2 = sort.Search(len(finalNodes)-1, func(k int) bool { return path[idx+1] <= finalNodes[k].id })
				coords = append(coords, LeafletEdge{[]float32{finalNodes[path[idx].idx].lat, finalNodes[path[idx].idx].lon, finalNodes[path[idx+1].idx].lat, finalNodes[path[idx+1].idx].lon}})
			}

		}

		var answer = convLeafletEdge2JSONDijkstra(coords, distance)

		w.Header().Set("Content-Type", "application/json")
		w.Write(answer)

	}

}

type Marker struct {
	Do   string  `json:"do"`
	Type string  `json:"type"`
	Lat  float32 `json:"lat"`
	Lon  float32 `json:"lon"`
}

func setMarker(w http.ResponseWriter, r *http.Request) {
	log.Println("received marker request")

	switch r.Method {
	case "GET":
		log.Println("there is no GET method for marker!")
	case "POST":
		decoder := json.NewDecoder(r.Body)
		var req Marker
		err := decoder.Decode(&req)
		if err != nil {
			panic(err)
		}

		if req.Type == "start" {
			start = finalNodes[findClosestNode(req.Lat, req.Lon, 5)]
		} else if req.Type == "finish" {
			finish = finalNodes[findClosestNode(req.Lat, req.Lon, 5)]
		} else {
			log.Println("invalid marker type!")

		}
		log.Println("handled marker request.")

	}
}

func webHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "index.html")
	case "POST":
		log.Println(r.Body)
		decoder := json.NewDecoder(r.Body)
		var resp Command
		err := decoder.Decode(&resp)
		if err != nil {
			panic(err)
		}
		log.Println(resp.Do)
		log.Println(resp.NWtlLat)
		log.Println(resp.NWtlLon)

		log.Println(resp.SEbrLat)
		log.Println(resp.SEbrLon)

		//build answer here
		var answer = convLeafletEdge2JSON(filterEdges(resp.NWtlLat, resp.NWtlLon, resp.SEbrLat, resp.SEbrLon, resp.ZoomLevel))

		w.Header().Set("Content-Type", "application/json")
		w.Write(answer)

	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

type EdgeIn struct {
	//n1idx --> searchsorted
	n1idx    uint32
	distance uint16
	speed    uint8
	access   uint8
}

type EdgesIn struct {
	eIn []EdgeIn
}

func (data *EdgesIn) AppendOffer(edgeIn EdgeIn) {
	data.eIn = append(data.eIn, edgeIn)
}

func main() {
	defer profile.Start().Stop()
	startTime := time.Now()
	fmt.Println("start parsing")
	decoder()

	fmt.Println("building edges in")

	//finalEdgesIn = make(map[uint32]*EdgesIn)
	//finalEdgesIn = make(map[uint32][]EdgeIn)
	var edgeErrors = 0

	//finalEdgesIn = make([]Edge, len(finalEdgesOut))
	//for _, e := range finalEdgesOut {

	//}

	//for _, e := range finalEdgesOut {
	//	//searchsorted n1index
	//	// find e.n1 in finalNodes
	//
	//	//fmt.Println(len(finalEdgesIn))
	//	//fmt.Print("\r", len(finalEdgesIn), "edges scanned")
	//	var i1 = sort.Search(len(finalNodes)-1, func(k int) bool { return e.n1 <= finalNodes[k].id })
	//
	//	if finalNodes[i1].id == e.n1 {
	//		if _, ok := finalEdgesIn[e.n2idx]; ok {
	//
	//			ob := finalEdgesIn[e.n2idx]
	//			ob.AppendOffer(EdgeIn{uint32(i1), e.distance, e.speed, e.access})
	//			finalEdgesIn[e.n2idx] = ob
	//
	//		} else {
	//
	//			ob := &EdgesIn{[]EdgeIn{}}
	//			ob.AppendOffer(EdgeIn{uint32(i1), e.distance, e.speed, e.access})
	//			finalEdgesIn[e.n2idx] = ob
	//		}
	//	} else {
	//		edgeErrors++
	//		log.Printf("no edge in found!")
	//	}
	//
	//}
	elapsed := time.Since(startTime)
	log.Printf("\rfinished parsing graph and building edgesIn in %s", elapsed)
	log.Printf("converted strange edges to points: %d", unusualEdges)
	log.Printf("no edges in for: %d", edgeErrors)

	//startup webserver
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/web/", http.StripPrefix("/web/", fs))

	http.HandleFunc("/", webHandler)
	http.HandleFunc("/dijkstra", dijkstra)
	http.HandleFunc("/marker", setMarker)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
