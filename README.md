# OgoSMos

## Install
Requirements:
- Ubuntu 18.04.03 LTS
- go 1.13.4 linux/amd64
- go package : github.com/thomersch/gosmparse
- A OSM-file of germany named "germany.osm.pbf"

1. If not already installed, install the go language: https://golang.org/
2. Clone this repository and unzip it to a folder of your choice
3. Set the go-workspace to the directory where you extracted the repo (https://github.com/golang/go/wiki/SettingGOPATH#go-113)
4. Navigate to `yourFolder/src/github.com/hulaalol/gosm/` in the command-line and run `go get -u github.com/thomersch/gosmparse` to install the OSM-Parser
5. Place the map file `germany.osm.pbf` in the data folder of the repo.
6. Now you can run `go run start.go` in the directory `yourFolder/src/github.com/hulaalol/gosm/`.

During parsing, the program might need about 13-14GB of Memory for a short time, so if your machine has only 16GB memory close some Chrome-Tabs ;)

## Usage
![1](/src/github.com/hulaalol/gosm/doc/1.png)
![2](/src/github.com/hulaalol/gosm/doc/2.png) 
![3](/src/github.com/hulaalol/gosm/doc/3.png) 
![4](/src/github.com/hulaalol/gosm/doc/4.png) 
![5](/src/github.com/hulaalol/gosm/doc/5.png) 
