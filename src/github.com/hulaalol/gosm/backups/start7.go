package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
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
const nofEdges = 70000000

var eMutex sync.Mutex

var finalNodes []Node
var finalEdgesOut []Edge
var finalOffsetsOut []uint32

//var finalOffsetsIn1 []uint32
//var finalOffsetsIn2 []uint32

// geodistance function

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
	n2Idx    uint32
	distance uint16
	speed    uint8
	access   uint8
}

type dataHandlerNodes struct {
	nodeCount uint32
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
		d.nodes[d.nodeCount] = Node{n.Element.ID, d.nodeCount, float32(n.Lat), float32(n.Lon)}
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
	start := time.Now()

	file := "data/stgt.osm.pbf"
	//file := "data/germany.osm.pbf"

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
	b := [3]string{"bridleway", "escape", "raceway"}
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
	f := [2]string{"footway", "steps"}
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
	//runtime.GC()

	var idx = 0
	for i := len(nodes2) - 1; i >= 0; i-- {
		if nodes2[i].id > 0 {
			idx = i
			log.Printf("last node %d", idx)
			break
		}
	}

	nodes3 := make([]Node, idx)
	copy(nodes3, nodes2[0:idx])

	idx = 0
	for i := len(edgesOut) - 1; i >= 0; i-- {
		if edgesOut[i].n1 > 0 {
			idx = i
			log.Printf("last edge %d", idx)
			break
		}
	}

	edgesOut2 := make([]Edge, idx)
	copy(edgesOut2, edgesOut[0:idx])

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

	//offsetsIn1 := make([]uint32, 0)
	//offsetsIn2 := make([]uint32, 0)

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
				//offsetsIn1 = append(offsetsIn1, uint32(i))
				//offsetsIn2 = append(offsetsIn2, uint32(c2I))
				var c2Lon = nodes3[c2I].lon
				var c2Lat = nodes3[c2I].lat
				edgesOut2[currOffset].distance = geoDistance(c1Lat, c1Lon, c2Lat, c2Lon)
			}
			currOffset = currOffset + 1

			if currOffset == uint32(len(edgesOut2)) {
				break
			}
		}
		offsets = append(offsets, currOffset)
	}

	finalNodes = nodes3

	for index, element := range finalNodes {
		element.idx = uint32(index)
	}

	finalEdgesOut = edgesOut2

	for index, element := range finalEdgesOut {
		// search in sorted array nodes3.id the value of edgesOut2[currOffset].n2

		//searching n2Index for edge
		var nI = uint32(sort.Search(len(finalNodes)-1, func(k int) bool { return element.n2 <= finalNodes[k].id }))
		finalEdgesOut[index].n2Idx = nI
	}

	finalOffsetsOut = offsets
	//finalOffsetsIn1 = offsetsIn1
	//finalOffsetsIn2 = offsetsIn2

	elapsed := time.Since(start)
	log.Printf("\rfinished parsing and building graph in %s", elapsed)

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
		fmt.Println("\r")
		//fmt.Printf("%v", offsetsIn1[0:n])
		//fmt.Printf("%v", offsetsIn2[0:n])
		fmt.Println("\r")

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
type path struct {
	value uint32
	nodes []uint32
}

type minPath []path

func (h minPath) Len() int           { return len(h) }
func (h minPath) Less(i, j int) bool { return h[i].value < h[j].value }
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

func getPath(origin uint32, destiny uint32) (uint32, []uint32) {

	h := newHeap()
	h.push(path{value: 0, nodes: []uint32{origin}})

	visited := make(map[uint32]bool)

	for len(*h.values) > 0 {
		// Find the nearest yet to visit node
		p := h.pop()
		node := p.nodes[len(p.nodes)-1]

		if visited[node] {
			continue
		}

		if node == destiny {
			return p.value, p.nodes
		}

		//get edges for this node
		//fmt.Println("nodeIdx")
		//fmt.Println(nodeIdx)

		// get eOut
		var eOutIdxStart = finalOffsetsOut[node]
		var eOutIdxEnd = finalOffsetsOut[node+1]
		var eOut = finalEdgesOut[eOutIdxStart:eOutIdxEnd]

		//get eIn
		//var edgeInIdx = sort.Search(len(finalOffsetsIn1)-1, func(k int) bool { return uint32(nodeIdx) <= finalOffsetsIn1[k] })
		////fmt.Println("edgeInIdx")
		////fmt.Println(edgeInIdx)
		//for finalEdgesOut[finalOffsetsIn2[edgeInIdx]].n2 == finalNodes[nodeIdx].id {
		//	var tmp = finalEdgesOut[finalOffsetsIn2[edgeInIdx]]
		//	eOut = append(eOut, Edge{tmp.n2, tmp.n1, tmp.distance, tmp.speed, tmp.access})
		//	edgeInIdx++

		//}

		for _, e := range eOut {
			if !visited[e.n2Idx] {
				// We calculate the total spent so far plus the cost and the path of getting here
				h.push(path{value: p.value + uint32(e.distance), nodes: append([]uint32{}, append(p.nodes, e.n2Idx)...)})
			}
		}

		visited[node] = true
	}
	return 0, []uint32{0}
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
		log.Println(r.Body)
		decoder := json.NewDecoder(r.Body)
		var req DijkstraInput
		err := decoder.Decode(&req)
		if err != nil {
			panic(err)
		}
		log.Println(req.Do)
		log.Println(req.Mode)

		//build answer here

		var startNode = finalNodes[findClosestNode(req.StartLat, req.StartLon, 10)]
		log.Println(startNode)

		var destNode = finalNodes[findClosestNode(req.TargetLat, req.TargetLon, 10)]
		log.Println(destNode)

		var distance, path = getPath(startNode.idx, destNode.idx)

		var coords = make([]LeafletEdge, 0)

		for idx, _ := range path {

			if idx != len(path)-1 {

				//var nodeIdx1 = sort.Search(len(finalNodes)-1, func(k int) bool { return path[idx] <= finalNodes[k].idx })
				//var nodeIdx2 = sort.Search(len(finalNodes)-1, func(k int) bool { return path[idx+1] <= finalNodes[k].idx })
				coords = append(coords, LeafletEdge{[]float32{finalNodes[path[idx]].lat, finalNodes[path[idx]].lon, finalNodes[path[idx+1]].lat, finalNodes[path[idx+1]].lon}})
			}

		}

		var answer = convLeafletEdge2JSONDijkstra(coords, distance)

		w.Header().Set("Content-Type", "application/json")
		w.Write(answer)

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
		//
		//		profile := Profile{"Alex", []string{"snowboarding", "programming"}}
		//
		//		js, err := json.Marshal(profile)
		//		if err != nil {
		//			http.Error(w, err.Error(), http.StatusInternalServerError)
		//			return
		//		}
		//
		w.Header().Set("Content-Type", "application/json")
		w.Write(answer)

		// Call ParseForm() to parse the raw query and update r.PostForm and r.Form.
		//if err := r.ParseForm(); err != nil {
		//	fmt.Fprintf(w, "ParseForm() err: %v", err)
		//	return
		//}
		//fmt.Fprintf(w, "Post from website! r.PostFrom = %v\n", r.PostForm)
		//name := r.FormValue("name")
		//address := r.FormValue("address")
		//fmt.Fprintf(w, "Name = %s\n", name)
		//fmt.Fprintf(w, "Address = %s\n", address)
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

type NodeCrumb struct {
	in  []uint32
	out []uint32
}

func main() {
	fmt.Println("start parsing")
	decoder()

	//var nodeRef = make(map[int64]NodeCrumb)

	//for idx, _ := range finalNodes {
	//
	//}

	//startup webserver
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/web/", http.StripPrefix("/web/", fs))

	http.HandleFunc("/", webHandler)
	http.HandleFunc("/dijkstra", dijkstra)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
