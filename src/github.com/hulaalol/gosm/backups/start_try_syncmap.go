package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/thomersch/gosmparse"
)

// Implement the gosmparser.OSMReader interface here.
// Streaming data will call those functions.
const nofNodes = 500000000
const nofNodes2 = 70000000
const nofEdges = 70000000

type Empty struct {
}

type Loc struct {
	lat float32
	lon float32
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
	distance float32
}

type dataHandlerNodes struct {
	nodeCount int
	nodes     *sync.Map
	mutex     *sync.Mutex
}

type dataHandlerWays struct {
	nodeCount int
	wayCount  int
	nodes     *sync.Map
	edgesOut  *[nofEdges]Edge
	mutex     *sync.Mutex
}

type dataHandlerRelations struct {
}

// data handler for nodes
func (d *dataHandlerNodes) ReadNode(n gosmparse.Node) {
	result, ok := d.nodes.Load(n.Element.ID)
	if ok {
		d.nodes.Store(n.Element.ID, Loc{float32(n.Lat), float32(n.Lon)})
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
			d.nodes.Store(w.NodeIDs[i], Loc{0, 0})
			d.nodes.Store(w.NodeIDs[i+1], Loc{0, 0})
			d.nodeCount = d.nodeCount + 2
		}
		d.mutex.Unlock()

	}

}
func (d *dataHandlerWays) ReadRelation(r gosmparse.Relation) {

}

func decoder() {
	start := time.Now()

	//file := "stgt.osm.pbf"
	file := "germany.osm.pbf"

	fmt.Println("parsing edges ...")
	r, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	dec := gosmparse.NewDecoder(r)
	// Parse will block until it is done or an error occurs.
	var nodes sync.Map
	var edgesOut [nofEdges]Edge

	d := dataHandlerWays{0, 0, &nodes, &edgesOut, &sync.Mutex{}}

	//err = dec.Parse(&dataHandler{})
	err = dec.Parse(&d)

	fmt.Println("get nodes location")
	r, err = os.Open(file)

	d2 := dataHandlerNodes{0, &nodes, &sync.Mutex{}}

	dec = gosmparse.NewDecoder(r)
	err = dec.Parse(&d2)

	var idx = 0
	for i := len(edgesOut) - 1; i >= 0; i-- {
		if edgesOut[i].n1 > 0 {
			idx = i
			log.Printf("last edge %d", idx)
			break
		}
	}
	var edgesOut2 = edgesOut[0:idx]

	fmt.Println("done ...")

	nodesList := make([]Node, d.nodeCount)

	i := 0
	nodes.Range(func(k, v) bool {
		nodesList[i] = Node{k, v.lat, v.lon}
		fmt.Println("key:", k, ", val:", v)
		return true
		// if false, Range stops
	})

	for n := range nodes {
		nodesList = append(nodesList, n)
	}

	fmt.Println("sorting nodes")
	sort.Slice(nodes3[:], func(i, j int) bool {
		return nodes3[i].id < nodes3[j].id
	})

	fmt.Println("sorting edges")
	sort.Slice(edgesOut2[:], func(i, j int) bool {
		return edgesOut2[i].n1 < edgesOut2[j].n1
	})

	fmt.Println(len(nodes3))
	fmt.Println(d.wayCount)

	fmt.Printf("%v", nodes3[0:15])
	fmt.Println("\r")
	fmt.Printf("%v", edgesOut2[0:15])
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
