package main

import (
	hp "container/heap"
	"log"
)

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
	streetname  string
}

type SNode struct {
	idx        uint32
	lat        float32
	lon        float32
	streetname string
}

func getEdges(node Node) []SNode {
	var eOutIdxStart = finalOffsetsOut[node.idx]
	var eOutIdxEnd = finalOffsetsOut[node.idx+1]
	var eOut = finalEdgesOut[eOutIdxStart:eOutIdxEnd]
	var result = make([]SNode, len(eOut))

	for idx, e := range eOut {
		result[idx] = SNode{e.n2idx, finalNodes[e.n2idx].lat, finalNodes[e.n2idx].lon, e.streetname}
	}
	return result
}

func getQuizPath(origin Node, destiny Node, mode string, metric string) (uint32, []SNode, []SNode) {
	var distance, path = getPath(origin, destiny, mode, metric)
	var edgeOptions = getEdges(origin)
	return distance, path, edgeOptions
}

func getPath(origin Node, destiny Node, mode string, metric string) (uint32, []SNode) {

	h := newHeap()
	h.push(path{value: 0, heuristic: 0, node: origin.idx})

	t2 := make([]tracker, len(finalNodes))
	t2[start.idx] = tracker{0, 0, ""}
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
			snames := make([]string, 0)

			//for c != origin.idx {
			for true {
				path = append(path, c)
				snames = append(snames, t2[c].streetname)

				if c == start.idx {
					break
				}

				c = t2[c].came_from
			}

			var result = make([]SNode, len(path))
			for idx, n := range path {
				result[idx] = SNode{n, finalNodes[n].lat, finalNodes[n].lon, snames[idx]}
			}
			result = append([]SNode{}, result...)
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
				t2[e.n2idx] = tracker{came_from, cost_so_far, e.streetname}
			}

		}
	}
	log.Printf("can't find a way :(, try again")
	return 0, make([]SNode, 0)
}
