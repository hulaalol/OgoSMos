# OgoSMos

## Install
Requirements:
- Ubuntu 18.04.03 LTS
- go 1.13.4 linux/amd64
- go package : github.com/thomersch/gosmparse
- An OSM-file of germany named "germany.osm.pbf"

1. If not already installed, install the go language: https://golang.org/
2. Clone this repository and unzip it to a folder of your choice. (Named `yourFolder` in the following steps)
3. Navigate to `yourFolder/src/github.com/hulaalol/gosm/` in the command-line and run `go get -u github.com/thomersch/gosmparse` to install the OSM-Parser
4. Create the data folder `yourFolder/src/github.com/hulaalol/gosm/data`.
5. Place the map file `germany.osm.pbf` in the data folder of the repo.
6. Now you can run `go run start.go` in the directory `yourFolder/src/github.com/hulaalol/gosm/`.

During parsing, the program might need about 13-14GB of Memory for a short time, so if your machine has only 16GB memory close some Chrome-Tabs ;)

## Benchmark
The program was run on a i7-4700MQ @ 2.8GHz, with Hyper-Threading enabled (4C/8T).
The machine had 16GB of memory.  

Parsing Time: 3min 23s  
Route Stuttgart-Hamburg, Car, Shortest Distance: 10.85 s  

## Usage
After parsing is done, a web server is running the GUI at `localhost:8080/web/`.
The GUI is very minimalistic. When you access the site, the german road network is generated.
Please allow some time for the leaflet.js framework to draw the layer on top of the map.
![1](/src/github.com/hulaalol/gosm/doc/1.png)
### Visualizing street graph
If you zoom out, at a certain zoomlevel only big roads are shown:
![2](/src/github.com/hulaalol/gosm/doc/2.png) 
The zoom process takes some time, because leaflet is rendering the data on a canvas.

Whenever you see the small rotating circle in the top right, the GUI has made a request to the server and is waiting for an answer. So don't press any buttons while the server is handling the request :)
![3](/src/github.com/hulaalol/gosm/doc/3.png) 

### Routing
Before calculating the path, please deactivate the street graph by clicking "Hide Graph" (because the germany layer of leaflet.js takes a big chunk of memory).

Next you can select which travelmode you want to select: Car, Cyclist or Pedestrian.
You can select if the program should search for shortest distance or shortest traveltime.

In the next step you have to click the map to place the start and finish markers.
The program will search for the closest node which respects your mode of travel and snap the marker to it. 
Finally, you can click calculate route to get the shortest path:
![4](/src/github.com/hulaalol/gosm/doc/4.png) 

![5](/src/github.com/hulaalol/gosm/doc/5.png) 
