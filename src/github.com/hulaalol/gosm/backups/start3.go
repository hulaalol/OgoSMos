package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
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
}

// data handler for nodes
func (d *dataHandlerNodes) ReadNode(n gosmparse.Node) {
	if _, ok1 := (*d.nodesDict)[n.Element.ID]; ok1 {
		d.mutex.Lock()
		d.nodes[d.nodeCount] = Node{n.Element.ID, float32(n.Lat), float32(n.Lon)}
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

	//file := "stgt.osm.pbf"
	file := "germany.osm.pbf"

	fmt.Println("parsing edges ...")
	r, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	dec := gosmparse.NewDecoder(r)
	// Parse will block until it is done or an error occurs.
	var nodes [nofNodes]int64
	var edgesOut [nofEdges]Edge

	d := dataHandlerWays{0, 0, &nodes, &edgesOut, &sync.Mutex{}}

	//err = dec.Parse(&dataHandler{})
	err = dec.Parse(&d)

	fmt.Println("building nodesDict ...")
	var nodesDict = make(map[int64]Empty)
	for i := 0; i < len(nodes)-1; i++ {
		nodesDict[nodes[i]] = Empty{}
	}
	runtime.GC()

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
