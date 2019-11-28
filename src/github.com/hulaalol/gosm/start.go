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

type LeafletMarker struct {
	Name string
	Lat  float32
	Lon  float32
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
	log.Printf("start filtering edges")

	var e = make([]LeafletEdge, 0)

	for index, element := range finalNodes {
		if element.lat < NWtlLat && element.lat > SEbrLat && element.lon > NWtlLon && element.lon < SEbrLon {

			var e0 = finalEdgesOut[finalOffsetsOut[index]:finalOffsetsOut[index+1]]

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
			}
		}
	}
	log.Printf("%d%s", len(e), "edges are visible on the map:")
	return e

}

func convLeafletEdge2JSONDijkstra(data []LeafletEdge, distance uint32) []byte {

	log.Printf("%s%d", "len of json-map data to convert: ", len(data))

	profile := LeafletEdgeArrayDijkstra{"edges", data, distance}
	js, err := json.Marshal(profile)
	if err != nil {
		log.Printf("error while converting LeafletEdgeArray to JSON")
		return []byte{0}
	}
	return js
}

func convLeafletEdge2JSON(data []LeafletEdge) []byte {

	log.Printf("%s%d", "len of json-path data to convert: ", len(data))

	profile := LeafletEdgeArray{"edges", data}
	js, err := json.Marshal(profile)
	if err != nil {
		log.Printf("error while converting LeafletEdgeArray to JSON")
		return []byte{0}
	}
	return js
}

// dijkstra
// heap implementation
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

func findClosestNode(lat float32, lon float32, delta uint16, mode string) int {
	var minIdx = 0
	var minD = 999999999.0

	var cDelta = float32(0.00001)

	for idx, node := range finalNodes {

		if node.lat-lat > cDelta || node.lon-lon > cDelta {
			continue
		}

		var d = geoDistancePrecision(lat, lon, node.lat, node.lon)

		if d < float64(delta) {

			e := finalEdgesOut[finalOffsetsOut[idx]:finalOffsetsOut[idx+1]]

			var found = false
			if mode == "car" {
				for _, edge := range e {
					if edge.access <= 1 {
						found = true
					}
				}
			}

			if mode == "bike" {
				for _, edge := range e {
					if edge.access != 1 && edge.access != 4 {
						found = true
					}
				}
			}

			if mode == "pedestrian" {
				for _, edge := range e {
					if edge.access != 1 && edge.access != 2 {
						found = true
					}
				}
			}

			if found == true {
				return idx
			}

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

func getPath(origin Node, destiny Node, mode string, metric string) (uint32, []Node) {

	h := newHeap()
	h.push(path{value: 0, heuristic: 0, node: origin.idx})

	t2 := make([]tracker, len(finalNodes))
	t2[start.idx] = tracker{0, 0}
	var new_cost uint32
	var currentNode tracker
	var heureka float64

	for len(*h.values) > 0 {
		// Find the node with the highest prio
		p := h.pop()
		node := p.node

		if node == destiny.idx {
			// return, we found the path
			c := destiny.idx
			path := make([]uint32, 0)

			for c != origin.idx {
				path = append(path, c)
				c = t2[c].came_from
			}
			var result = make([]Node, len(path))
			for idx, n := range path {
				result[idx] = Node{0, n, finalNodes[n].lat, finalNodes[n].lon}
			}
			return p.value, result
		}

		// collect edges
		var eOutIdxStart = finalOffsetsOut[node]
		var eOutIdxEnd = finalOffsetsOut[node+1]
		var eOut = finalEdgesOut[eOutIdxStart:eOutIdxEnd]

		for _, e := range eOut {

			// skip edges not allowed for this mode of travel
			if mode == "car" && e.access > 1 {
				continue
			} else if mode == "bike" && (e.access == 1 || e.access == 4) {
				continue
			} else if mode == "pedestrian" && (e.access == 1 || e.access == 2) {
				continue
			}

			if metric == "time" && mode == "car" {
				new_cost = t2[node].cost_so_far + uint32(3600.0/((1000.0*float32(e.speed))/float32(e.distance)))
			} else {
				new_cost = t2[node].cost_so_far + uint32(e.distance)
			}

			exists := false

			if t2[e.n2idx].came_from != 0 {
				exists = true
				currentNode = t2[e.n2idx]
			}

			if !exists || (new_cost < currentNode.cost_so_far) {

				cost_so_far := new_cost

				if metric == "time" && mode == "car" {
					heureka = 3600.0 / (50000.0 / QuickDistance(destiny.lat, destiny.lon, finalNodes[e.n2idx].lat, finalNodes[e.n2idx].lon))
				} else {
					heureka = QuickDistance(destiny.lat, destiny.lon, finalNodes[e.n2idx].lat, finalNodes[e.n2idx].lon)
				}
				prio := float64(new_cost) + heureka
				h.push(path{value: new_cost, heuristic: prio, node: finalNodes[e.n2idx].idx})
				came_from := p.node
				t2[e.n2idx] = tracker{came_from, cost_so_far}
			}

		}
	}
	log.Printf("can't find a way :(, try again")
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
	Do     string `json:"do"`
	Mode   string `json:"mode"`
	Metric string `json:"metric"`
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

		startTime := time.Now()
		var distance, path = getPath(start, finish, req.Mode, req.Metric)

		if req.Metric == "distance" {
			log.Printf("found path with %d nodes and %d m length", len(path), distance)
		} else {
			log.Printf("found path with %d nodes and %d s travel duration", len(path), distance)
		}

		elapsed := time.Since(startTime)
		log.Printf("\rfinished finding dijkstra way in %s", elapsed)

		var coords = make([]LeafletEdge, 0)

		for idx, _ := range path {

			if idx != len(path)-1 {
				coords = append(coords, LeafletEdge{[]float32{finalNodes[path[idx].idx].lat, finalNodes[path[idx].idx].lon, finalNodes[path[idx+1].idx].lat, finalNodes[path[idx+1].idx].lon}})
			}

		}

		var answer = convLeafletEdge2JSONDijkstra(coords, distance)

		w.Header().Set("Content-Type", "application/json")
		w.Write(answer)

	}

}

type Marker struct {
	Type string  `json:"type"`
	Mode string  `json:"mode"`
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

		var jsonMarker LeafletMarker
		if req.Type == "start" {
			start = finalNodes[findClosestNode(req.Lat, req.Lon, 100, req.Mode)]

			jsonMarker = LeafletMarker{"start", start.lat, start.lon}

		} else if req.Type == "finish" {
			finish = finalNodes[findClosestNode(req.Lat, req.Lon, 100, req.Mode)]

			jsonMarker = LeafletMarker{"finish", finish.lat, finish.lon}

		} else {
			log.Println("invalid marker type!")

		}

		js, err := json.Marshal(jsonMarker)
		if err != nil {
			log.Printf("error while converting LeafletMarker to JSON")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
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

	//startup webserver
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/web/", http.StripPrefix("/web/", fs))

	http.HandleFunc("/", webHandler)
	http.HandleFunc("/dijkstra", dijkstra)
	http.HandleFunc("/marker", setMarker)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
