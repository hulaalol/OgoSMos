# QuizNav

## Install
# Requirements:
1. Ubuntu 18.04.03 LTS
2. Google Chrome Version 83.0
3. go 1.13.4 linux/amd64
4. go package : github.com/thomersch/gosmparse
5. An OSM-file of London named "london.osm.pbf" (already included in the data archive)

# Installation:
1. If not already installed, install the go language: https://golang.org/
2. Navigate to `quizNav/src/github.com/hulaalol/gosm/` in the command-line and run `go get -u github.com/thomersch/gosmparse` to install the OSM-Parser
3. Now you can run `go run start.go webserver.go osmParser.go routing.go questionmaster.go` in the directory `quizNav/src/github.com/hulaalol/gosm/`.
4. visit localhost:8080/web to access the game (Google Chrome with 1920x1080 pixel resoultion is the recommended and tested browser environment)

# How to play
This shipped version contains the London map file and works in the London Metropolitan area. 
It is possible to replace the map file and play in any other city around the globe.
However, cities with english street names are preferred because the dbpedia database contains the most knowledge for the english language.
Before you start the game you can set a start and finish location for your game round.
At the top of the screen in the center you can see in which street you are currently.
At the bottom of the screen the remaining distance to the finish line is shown.

## How to win
You are navigated to the finish location.
For every turn on your path a question is generated from the name of the street, which you will have to answer.
If you answer the question correctly you may stay on the correct path. Wrong answers will route you away from the optimal path.
Your goal is to reach the finish line with the shortest possible distance.
The perfect score is 100 points which you can achieve if you answer all questions correctly.

# Known issues:
### The depiction of items in the answer window may overlap into the continue-button hindering the player to continue the game:
Set your screen/browser resolution to 1920x1080 pixel. Use the Chrome Browser if possible.

### In rare cases no question is generated anymore and the game is "stuck":
The game encountered a circular reference in dbpedia:disambiguates properties and is captured in this loop of discovering disambiguations.
Sorry, you have to restart your round :(





