package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/thomersch/gosmparse"
)

// Implement the gosmparser.OSMReader interface here.
// Streaming data will call those functions.
const nofNodes = 450000000
const nofNodes2 = 55000000
const nofEdges = 55000000

type Empty struct {
}

type Node struct {
	id  int64
	lat float32
	lon float32
}

type Edge struct {
	n1       int64
	n2       int64
	speed    uint8
	distance uint16
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
	nodeCount int
	wayCount  int
	nodes     *[nofNodes]int64
	edgesOut  *[nofEdges]Edge
	mutex     *sync.Mutex
	blacklist *map[string]Empty
}

// data handler for nodes
func (d *dataHandlerNodes) ReadNode(n gosmparse.Node) {
	if _, ok1 := (*d.nodesDict)[n.Element.ID]; ok1 {
		d.mutex.Lock()
		d.nodes[d.nodeCount] = Node{n.Element.ID, float32(n.Lat), float32(n.Lon)}
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

		if _, ok3 := (*d.blacklist)[w.Element.Tags["highway"]]; ok3 {
			return
		}

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

		d.mutex.Lock()
		for i := 0; i < len(w.NodeIDs)-1; i++ {
			d.edgesOut[d.wayCount] = Edge{w.NodeIDs[i], w.NodeIDs[i+1], uint8(m), 0}
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

	file := "stgt.osm.pbf"
	//file := "germany.osm.pbf"

	fmt.Println("parsing edges ...")
	r, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	dec := gosmparse.NewDecoder(r)
	// Parse will block until it is done or an error occurs.
	var nodes [nofNodes]int64
	var edgesOut [nofEdges]Edge
	blacklist := make(map[string]Empty)
	b := [6]string{"track", "bus_guideway", "bridleway", "escape", "raceway", "steps"}
	for i := 0; i < len(b); i++ {
		blacklist[b[i]] = Empty{}
	}

	d := dataHandlerWays{0, 0, &nodes, &edgesOut, &sync.Mutex{}, &blacklist}

	//err = dec.Parse(&dataHandler{})
	err = dec.Parse(&d)

	fmt.Println("building nodesDict ...")
	var nodesDict = make(map[int64]Empty)
	for i := 0; i < len(nodes)-1; i++ {
		nodesDict[nodes[i]] = Empty{}
	}

	debug.FreeOSMemory()
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
	var nodes3 = nodes2[0:idx]

	idx = 0
	for i := len(edgesOut) - 1; i >= 0; i-- {
		if edgesOut[i].n1 > 0 {
			idx = i
			log.Printf("last edge %d", idx)
			break
		}
	}
	var edgesOut2 = edgesOut[0:idx]

	fmt.Println("done ...")

	fmt.Println("sorting nodes")
	sort.Slice(nodes3[:], func(i, j int) bool {
		return nodes3[i].id < nodes3[j].id
	})

	fmt.Println("sorting edges")
	sort.Slice(edgesOut2[:], func(i, j int) bool {
		return edgesOut2[i].n1 < edgesOut2[j].n1
	})

	offsets := make([]uint32, 0)
	offsets = append(offsets, 0)

	for i := 1; i <= len(nodes3)-1; i++ {

		if nodes3[i].id < edgesOut2[offsets[i-1]].n1 {
			offsets = append(offsets, offsets[i-1])
			break
		}
		var currOffset = offsets[i-1]
		for nodes3[i-1].id == edgesOut2[currOffset].n1 {
			var c1Lon = nodes3[i-1].lon
			var c1Lat = nodes3[i-1].lat
			var c2I = sort.Search(len(nodes3)-1, func(k int) bool { return (int(edgesOut2[currOffset].n2) == int(nodes3[k].id)) })

			if edgesOut2[currOffset].n1 == nodes3[c2I].id {
				var c2Lon = nodes3[c2I].lon
				var c2Lat = nodes3[c2I].lat

				edgesOut[currOffset].distance = uint16(c1Lon + c1Lat + c2Lon + c2Lat)
			}
			currOffset = currOffset + 1
		}
		offsets = append(offsets, currOffset)
	}

	fmt.Println(len(nodes3))
	fmt.Println(d.wayCount)

	fmt.Printf("%v", nodes3[0:15])
	fmt.Println("\r")
	fmt.Printf("%v", edgesOut2[0:15])
	fmt.Println("\r")
	fmt.Printf("%v", offsets[0:15])
	fmt.Println("\r")
	elapsed := time.Since(start)
	log.Printf("\rfinished in %s", elapsed)

	if err != nil {
		panic(err)
	}

}

func main() {
	fmt.Println("start parsing")
	decoder()
}
