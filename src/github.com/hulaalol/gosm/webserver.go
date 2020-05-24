package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"
)

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

type LeafletMarker struct {
	Name string
	Lat  float32
	Lon  float32
}

type LeafletEdge struct {
	// C are the 4 coordinates lat,lon of start and end of edge
	C []float32
	N string
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

type QuestionJS struct {
	Item   string
	Answer string
	D1     string
	D2     string
	D3     string
}

type QuizCrumb struct {
	Name                    string
	CurrentPos              []LeafletEdge
	DistractorEdges         [][]LeafletEdge
	DistanceToTarget        uint32
	Question                []QuestionJS
	Img                     string
	Abstract                string
	CurrentPosDistance      float64
	DistractorEdgesDistance []float64
}

//function to filter out edges for a certain zoom level
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

				e = append(e, LeafletEdge{[]float32{finalNodes[i1].lat, finalNodes[i1].lon, finalNodes[i2].lat, finalNodes[i2].lon}, ""})
			}
		}
	}
	log.Printf("%d%s", len(e), "edges are visible on the map:")
	return e

}

func convQuizCrumb2JSON(path []LeafletEdge, distance uint32, edgeOptions [][]LeafletEdge, qw QuestionWrapper, cDis float64, dDis []float64) []byte {

	//var qjs = make([]QuestionJS, len(questions))
	//for idx, q := range questions {
	//	qjs[idx] = QuestionJS{q.item, q.answer, q.d1, q.d2, q.d3}
	//}

	var qjs = []QuestionJS{QuestionJS{qw.question.item, qw.question.answer, qw.question.d1, qw.question.d2, qw.question.d3}}

	profile := QuizCrumb{"quiz", path, edgeOptions, distance, qjs, qw.img, qw.abstract, cDis, dDis}
	js, err := json.Marshal(profile)
	if err != nil {
		log.Printf("error while converting LeafletEdgeArray to JSON")
		return []byte{0}
	}
	return js
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

func reversePath(s []SNode) []SNode {

	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func quizNav(w http.ResponseWriter, r *http.Request) {

	log.Println("received request for quiz navigation")

	switch r.Method {

	case "GET":
		log.Println("there is no GET method for quiz navigation!")

	case "POST":
		decoder := json.NewDecoder(r.Body)
		var req DijkstraInput
		err := decoder.Decode(&req)
		if err != nil {
			panic(err)
		}

		var distance, path, edgeOptions = getQuizPath(start, finish, "car", "distance")

		var radius uint16 = 1
		for len(path) == 0 {
			fmt.Println("webserver request failed to generate path...")
			start = finalNodes[findClosestNode(start.lat, start.lon, radius*10, "car")]
			radius += 1
			distance, path, edgeOptions = getQuizPath(start, finish, "car", "distance")
		}

		if len(path) != 0 {

			// reverse path
			path = reversePath(path)

			var currentStreet = path[0].streetname
			fmt.Println("currently in street: " + currentStreet)
			var qw = genQuestion(currentStreet)
			var coords = make([]LeafletEdge, 0)
			var eOpts = make([][]LeafletEdge, 0)

			// build correct path until next junction
			var coordsDistance = 0.0
			for idx, _ := range path {

				// abort if finished navigating current street
				if path[idx].streetname != currentStreet {
					break
				}

				if idx != len(path)-1 {
					//coords = append(coords, LeafletEdge{[]float32{finalNodes[path[idx].idx].lat, finalNodes[path[idx].idx].lon, finalNodes[path[idx+1].idx].lat, finalNodes[path[idx+1].idx].lon}, path[idx].streetname})
					coordsDistance = coordsDistance + geoDistancePrecision(path[idx].lat, path[idx].lon, path[idx+1].lat, path[idx+1].lon)
					coords = append(coords, LeafletEdge{[]float32{path[idx].lat, path[idx].lon, path[idx+1].lat, path[idx+1].lon}, path[idx].streetname})

				}
			}

			var eOptsDistances = make([]float64, 0)
			// wrong turns if player fails to answer question
			for idx, _ := range edgeOptions {

				var tmp = make([]LeafletEdge, 0)
				var tmpDistance = 0.0
				for idx2, _ := range edgeOptions[idx] {

					//var k = true
					// dont use edges present in the path!
					//for idx3, _ := range path {
					//	if edgeOptions[idx][idx2].idx == path[idx3].idx {
					//		k = false
					//	}
					//}
					//if k {

					if idx2 != len(edgeOptions[idx])-1 {
						tmpDistance = tmpDistance + geoDistancePrecision(edgeOptions[idx][idx2].lat, edgeOptions[idx][idx2].lon, edgeOptions[idx][idx2+1].lat, edgeOptions[idx][idx2+1].lon)

						tmp = append(tmp, LeafletEdge{[]float32{edgeOptions[idx][idx2].lat, edgeOptions[idx][idx2].lon, edgeOptions[idx][idx2+1].lat, edgeOptions[idx][idx2+1].lon}, edgeOptions[idx][idx2].streetname})
						//tmp = append(tmp, LeafletEdge{[]float32{finalNodes[edgeOptions[idx][idx2].idx].lat, finalNodes[edgeOptions[idx][idx2].idx].lon, finalNodes[edgeOptions[idx][idx2+1].idx].lat, finalNodes[edgeOptions[idx][idx2+1].idx].lon}, edgeOptions[idx][idx2].streetname})

					}

					//eOpts = append(eOpts, LeafletEdge{[]float32{finalNodes[path[0].idx].lat, finalNodes[path[0].idx].lon, finalNodes[edgeOptions[idx].idx].lat, finalNodes[edgeOptions[idx].idx].lon}, edgeOptions[idx].streetname})

					//}

				}
				eOpts = append(eOpts, tmp)
				eOptsDistances = append(eOptsDistances, tmpDistance)

			}

			var answer = convQuizCrumb2JSON(coords, distance, eOpts, qw, coordsDistance, eOptsDistances)

			w.Header().Set("Content-Type", "application/json")
			w.Write(answer)

		} else {

			fmt.Println("webserver request failed to generate path...")
		}

	}

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
				coords = append(coords, LeafletEdge{[]float32{finalNodes[path[idx].idx].lat, finalNodes[path[idx].idx].lon, finalNodes[path[idx+1].idx].lat, finalNodes[path[idx+1].idx].lon}, path[idx].streetname})
			}

		}

		var answer = convLeafletEdge2JSONDijkstra(coords, distance)

		//fmt.Println(answer)
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

	var nodeDelta uint16 = 10

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
			start = finalNodes[findClosestNode(req.Lat, req.Lon, nodeDelta, req.Mode)]

			jsonMarker = LeafletMarker{"start", start.lat, start.lon}

		} else if req.Type == "finish" {
			finish = finalNodes[findClosestNode(req.Lat, req.Lon, nodeDelta, req.Mode)]

			jsonMarker = LeafletMarker{"finish", finish.lat, finish.lon}

		} else if req.Type == "loc" {
			start = finalNodes[findClosestNode(req.Lat, req.Lon, 1, req.Mode)]

			jsonMarker = LeafletMarker{"start", start.lat, start.lon}

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

func startWebserver() {

	//startup webserver
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/web/", http.StripPrefix("/web/", fs))

	http.HandleFunc("/", webHandler)
	http.HandleFunc("/dijkstra", dijkstra)
	http.HandleFunc("/marker", setMarker)
	http.HandleFunc("/quizNav", quizNav)

	log.Fatal(http.ListenAndServe(":8080", nil))

}
