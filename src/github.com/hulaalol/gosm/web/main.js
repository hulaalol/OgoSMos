


var markerSet = false;
var marker1;
var marker2;

//var autoUpdate = false;
var mode;
var metric;
var mymap;


var germanyLayer;
var graphHidden;


// quiznav globals
var gameStarted = false;
var globalQuestion;

var distanceScore;
var playerScore = 0;


// #4CAF50

function clearMap(m) {
    for(i in m._layers) {
        if(m._layers[i]._path != undefined) {
            try {
                m.removeLayer(m._layers[i]);
            }
            catch(e) {
                console.log("problem with " + e + m._layers[i]);
            }
        }
    }
}


async function initMap(map){

    activateLoader();
    // lock map
    document.getElementById("mapid").setAttribute("class", "locked");

    let promise = new Promise((resolve, reject) => {
        var xhr = new XMLHttpRequest();
        var url = "/";
        xhr.open("POST", url, true);
        xhr.setRequestHeader("Content-Type", "application/json");
    
        xhr.onreadystatechange = function () {
        if (xhr.readyState === 4 && xhr.status === 200) {
            //console.log(xhr.responseText);
            var json = JSON.parse(xhr.responseText);
            //console.log(json.Name);    
            var pl = []
            for (i = 0; i < json.Data.length; i++) {
                var edge = json.Data[i].C
                pl.push([[edge[0], edge[1]], [edge[2], edge[3]]])
            }

            var polyline = new L.polyline(pl, {smoothFactor:500, interactive: false})
            polyline.id = "germany"
            
            germanyLayer = polyline

            germanyLayer.addTo(mymap);

            mymap.invalidateSize()
            console.log("finished init map!")
            

            //unlock map
            document.getElementById("mapid").setAttribute("class", "unlocked");
            //toggleMapUpdate()
            deactivateLoader();
            resolve(1)
        }
        }

        var data = JSON.stringify({"do": "giveEdges",
                                    "zoomLevel": 0,
                                    "NWtlLat": 90, "NWtlLon": 0, 
                                    "SWblLat": 0, "SWblLon": 0,
                                    "SEbrLat": 0, "SEbrLon": 30,
                                    "NEtrLat": 90, "NEtrLon": 30, });
    
        console.log("sending json request to server: "+data)
        xhr.send(data);

      });
   
};

async function updateMap(map, bounds, zoomLevel){

    activateLoader();
    //clearMap(map)
    // lock map
    document.getElementById("mapid").setAttribute("class", "locked");

    let promise = new Promise((resolve, reject) => {
        var xhr = new XMLHttpRequest();
        var url = "/";
        xhr.open("POST", url, true);
        xhr.setRequestHeader("Content-Type", "application/json");
    
        xhr.onreadystatechange = function () {
        if (xhr.readyState === 4 && xhr.status === 200) {
            //console.log(xhr.responseText);
            var json = JSON.parse(xhr.responseText);
            //console.log(json.Name);    
            function smoothFactorCalc(z){
                return Math.round(1 + Math.exp(-1.3*z+16.5))
            }

            var sf = smoothFactorCalc(zoomLevel)

            var pl = []
            for (i = 0; i < json.Data.length; i++) {
                var edge = json.Data[i].C
                pl.push([[edge[0], edge[1]], [edge[2], edge[3]]])
            }

            var polyline = L.polyline(pl, {smoothFactor:sf, interactive: false})

            polyline.id = "detailZoom"
            polyline.addTo(mymap);


            map.invalidateSize()
            console.log("finished updating edges on map!")

            //unlock map
            document.getElementById("mapid").setAttribute("class", "unlocked");
            deactivateLoader();
            resolve(1)
        }
        }
        var NWtlLat = bounds.getNorthWest().lat
        var NWtlLon = bounds.getNorthWest().lng
        var SWblLat = bounds.getSouthWest().lat
        var SWblLon = bounds.getSouthWest().lng
        var SEbrLat = bounds.getSouthEast().lat
        var SEbrLon = bounds.getSouthEast().lng
        var NEtrLat = bounds.getNorthEast().lat
        var NEtrLon = bounds.getNorthEast().lng
        console.log("NWtlLat: "+NWtlLat)
    
    
        var data = JSON.stringify({"do": "giveEdges",
                                    "zoomLevel": zoomLevel,
                                    "NWtlLat": NWtlLat, "NWtlLon": NWtlLon, 
                                    "SWblLat": SWblLat, "SWblLon": SWblLon,
                                    "SEbrLat": SEbrLat, "SEbrLon": SEbrLon,
                                    "NEtrLat": NEtrLat, "NEtrLon": NEtrLon, });
    
        console.log("sending json request to server: "+data)
        xhr.send(data);

      });
   
};



$(document).ready(function(){


    //London = [51.505, -0.09]
    //51.515205894073105/-0.09278297424316408
    London = [51.515, -0.09]

    Stuttgart = [48.775, 9.1829321]

    document.getElementById("mapid").style.height = window.innerHeight;
    document.getElementById("mapid").style.width = window.innerWidth;

    mymap = L.map('mapid',{renderer: L.canvas()}).setView(London, 15);


    //selectCar();
    mode= "car";
    metric = "distance";
    graphHidden = true;
    //selectDistance();
    

    //mymap.on("moveend", async function () {
    mymap.on("moveend", async function () {


        mymap.eachLayer(function (layer) {
            if(layer.id && layer.id.includes("detailZoom")){
                mymap.removeLayer(layer);
            }
        });

        if(mymap.getZoom()<=13){
            return
        }

        //if(autoUpdate && !hideGraph){
        if(!graphHidden){

            var bounds = mymap.getBounds()
            //var loc = mymap.getCenter();
            //console.log(loc.toString());
            await updateMap(mymap, bounds, mymap.getZoom())
            console.log("fire update edges")
        }else{
            console.log("auto update is off")
        }


      });



    mymap.on('click', function(event){

        if(gameStarted){
            return
        }


        if(!markerSet){
            //set marker 1
            if(marker1){
                mymap.removeLayer(marker1)
            }
            marker1 = new L.marker(event.latlng)

             // send request to server

            marker1.bindTooltip("Start", 
            {
                permanent: true, 
                direction: 'right'
            })

            mymap.addLayer(marker1)
            sendMarker(marker1,"start")
            markerSet = true
        }else{

            if(marker2){
                mymap.removeLayer(marker2)
            }
            //set marker 2
            marker2 = new L.marker(event.latlng)

            // send request to server
            marker2.bindTooltip("Finish", 
            {
                permanent: true, 
                direction: 'right'
            })

            mymap.addLayer(marker2)
            sendMarker(marker2,"finish")
            markerSet = false
        }

    })

    mymap.on('zoomend', function() {
        //console.log("set current zoom to: "+mymap.getZoom())
    });


    L.tileLayer('https://api.tiles.mapbox.com/v4/{id}/{z}/{x}/{y}.png?access_token={accessToken}', {
        attribution: 'Map data &copy; <a href="https://www.openstreetmap.org/">OpenStreetMap</a> contributors, <a href="https://creativecommons.org/licenses/by-sa/2.0/">CC-BY-SA</a>, Imagery © <a href="https://www.mapbox.com/">Mapbox</a>',
        maxZoom: 20,
        id: 'mapbox.streets',
        accessToken: 'pk.eyJ1IjoiaHVsYWFsb2wiLCJhIjoiY2szYjBqc2Q3MGhuazNkbXhrbnZsaHIyYiJ9._R-lsR0wjcmJGXf5TRmSSw'
    }).addTo(mymap);


    initMap()

    // show welcome screen
    document.getElementById("startWindowContainer").style.visibility = "visible";

});


function sendMarker(marker,type){

    return new Promise((resolve,reject) => {
        activateLoader();
        var xhr = new XMLHttpRequest();
        var url = "/marker";
        xhr.open("POST", url, true);
        xhr.setRequestHeader("Content-Type", "application/json");
        
        xhr.onreadystatechange = function () {
        if (xhr.readyState === 4 && xhr.status === 200) {
    
            var json = JSON.parse(xhr.responseText);
            var newLatLng = new L.LatLng(json.Lat, json.Lon);
        
            if(json.Name == "start"){
                marker1.setLatLng(newLatLng); 
            }
            if(json.Name == "finish"){
                marker2.setLatLng(newLatLng); 
            }
    
            console.log("handled marker request :)");
            deactivateLoader();

            resolve(true);
    
        }
        }
        var lat = marker.getLatLng().lat
        var lon = marker.getLatLng().lng 
        var data = JSON.stringify({"do": "setMarker",
                                    "type" : type,
                                    "mode" : mode,
                                    "lat": lat,
                                    "lon": lon });
        
        console.log("sending marker to server: "+type+" - "+"("+lat+"/"+lon+")")
        xhr.send(data);
    });
}


/**
 * Shuffles array in place. ES6 version
 * @param {Array} a items An array containing the items.
 */
function shuffle(a) {
    for (let i = a.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [a[i], a[j]] = [a[j], a[i]];
    }
    return a;
}


function setMarker(latlng){
    return new Promise((resolve,reject) => {

        if(marker1){
            mymap.removeLayer(marker1)
        }
    
        marker1 = new L.marker(latlng)
         // send request to server
        marker1.bindTooltip("You", 
        {
            permanent: true, 
            direction: 'right'
        })
        mymap.addLayer(marker1)
        sendMarker(marker1,"loc").then(function(){
            resolve(true);
        }, function(){
            reject(true);
        })
    })

}


function continueGame(e){
        document.getElementById("answerWindowContainer").style.visibility = "hidden";
        document.getElementById("questionWindowContainer").style.visibility = "visible";
}

function startGame(e){

    if(marker1 && marker2){
        quizNav();
    }else{
        document.getElementById("startText").innerHTML  = "At least one marker is missing! Click the map to add markers.";
    }
}

function answerQuestion(e){

    e = e || window.event;
    var target= e.target || e.srcElement;
    
    if(target.id == globalQuestion.AnswerID){

        var audio = new Audio('correct.mp3');
        audio.play();


        //correct answer
        globalQuestion.cppl.addTo(mymap);

        var endOfPath = globalQuestion.cppl._latlngs.length;
        //var startC = globalQuestion.cppl._latlngs[0];
        var latlng = globalQuestion.cppl._latlngs[endOfPath-1][1];
        playerScore += globalQuestion.cppl.options.distance;

        
    }else{
        var audio = new Audio('wrong.mp3');
        audio.play();


        if(globalQuestion.wppls.length == 0){
            var wppl = globalQuestion.cppl;
            wppl.options.color = "red";
        }else{
            //incorrect answer
            var wppl= globalQuestion.wppls[Math.floor(Math.random() * globalQuestion.wppls.length)];
        }

            wppl.addTo(mymap);
            playerScore += wppl.options.distance;
            var endOfPath = wppl._latlngs.length;
            //var startC = wppl._latlngs[0];
            var latlng = wppl._latlngs[endOfPath-1][1];

    }


    // update score by adding distance of edge traveled
    //var geoDist = geoDistance(startC[0].lat, startC[0].lng, latlng.lat, latlng.lng);
    //playerScore += geoDist;

    console.log("Current player score is "+playerScore)
    mymap.setView([latlng.lat,latlng.lng],17);


    //update abstract
    
    //limit abstract
    var a = globalQuestion.abstract;
    if(a.length > 750){
        a = a.substring(0,750)
        var idx = a.lastIndexOf(".")
        a = a.substring(0,idx+1)
    }

    if(a.length > 10){
        document.getElementById("abstract").innerHTML = a;
    }else{
        document.getElementById("abstract").innerHTML = "No information was found :(";

    }


        //update image
        //var dep = document.getElementById("depiction");

        //var cWidth = document.getElementById("depictionCell").offsetWidth;
    
        //dep.style.maxWidth = cWidth+"px";

        //dep.style.maxHeight = cHeight-10+"px";
        //dep.style.minHeight = cHeight-10+"px";

        if(document.getElementById("depiction")){
            document.getElementById("depiction").outerHTML = "";
        }

        if(globalQuestion.img.length > 0 && globalQuestion.img != "null"){
            document.getElementById("answerWindowContainer").style.height ="55%";

            var cHeight = document.getElementById("depictionRow").offsetHeight;

            var x = document.createElement("IMG");
            x.setAttribute("src", globalQuestion.img);
            x.setAttribute("id","depiction");

            //x.style.maxHeight = cHeight+"px";
            //x.style.minHeight = cHeight+"px";

            document.getElementById("depictionCell").appendChild(x);

            //dep.src = globalQuestion.img;
            //dep.style.visibility = "visible";
        }else{
            document.getElementById("answerWindowContainer").style.height ="30%";
            

           

            //dep.style.visibility = "hidden";
        }



    document.getElementById("questionWindowContainer").style.visibility = "hidden";
    document.getElementById("answerWindowContainer").style.visibility = "visible";

    latdist = Math.abs(latlng.lat - marker2._latlng.lat);
    lngdist = Math.abs(latlng.lng - marker2._latlng.lng);
    var delta = 0.0001;

        setMarker(latlng).then(function(){
            //fullfillment
            if(latdist < delta && lngdist < delta){

                // calculate score

                var score = Math.round( 100*(1- Math.abs(1- (distanceScore/playerScore))) );
                //var score = 100 - (((distanceScore/(playerScore*1000))-1)*100);
                document.getElementById("answerWindowContainer").style.visibility = "hidden";
                document.getElementById("finishText").innerHTML = "Awesome! You made it to the finish line!<br>The shortest possible distance was "+distanceScore+" meters - you needed "+Math.round(playerScore)+" meters to reach the finish line.<br>Your score is "+score+" Points.";
                document.getElementById("finishWindowContainer").style.visibility = "visible";
                console.log("GAME OVER")
            }else{
                quizNav();   
            }
           }, function(reason){
               //rejection
           });
    

}


function setQuestion(question){

    document.getElementById("question").innerHTML = question.Item;

    var answers = [question.Answer,question.D1,question.D2,question.D3];
    var aIdx = ['1','2','3','4'];
    aIdx = shuffle(aIdx);

    for(var i=0;i<aIdx.length;i++){
        document.getElementById("a"+aIdx[i]).innerHTML = answers[i];
        //document.getElementById("a"+aIdx[i]).fontWeight="normal";
    }
    //document.getElementById("a"+aIdx[0]).style.fontWeight="bold";

    globalQuestion = question;
    globalQuestion.AnswerID = "a"+aIdx[0];
}

function pickQuestion(questions){
    var q= questions[Math.floor(Math.random() * questions.length)];
    if(q.Item == "null"){
        return pickQuestion(questions)
    }else{
        return q;
    }
}

function checkQuestions(questions){
    for(i=0; i<questions.length;i++){
        if(questions[i].Item != "null"){
            return true;
        }
    }
    return false;
}



function quizNav(){
    activateLoader();
    console.log("map cleaned...")


    let promise = new Promise((resolve, reject) => {

        document.getElementById("mapid").setAttribute("class", "locked");

        var xhr = new XMLHttpRequest();
        var url = "/quizNav";
        xhr.open("POST", url, true);
        xhr.setRequestHeader("Content-Type", "application/json");
    
        xhr.onreadystatechange = function () {
        if (xhr.readyState === 4 && xhr.status === 200) {
            

            germanyLayer = null;
            if(!graphHidden){
                hideGraph();
            }
        
            mymap.eachLayer(function (layer) {
                if(layer.id && layer.id.includes("path")){
                    mymap.removeLayer(layer);
                }
            });

            var json = JSON.parse(xhr.responseText);


            if(json.Distance == 0){
                document.getElementById("routeinfo").innerHTML = "Ops! Couldn't find way?! Check if the markers are placed in a lake, shopping centre or military base :)";

            }else{

                var correctPath = []
                for(i = 0; i< json.CurrentPos.length; i++){
                    var edge = json.CurrentPos[i].C;
                    correctPath.push([[edge[0], edge[1]], [edge[2], edge[3]]]);
                }

                var cppl = L.polyline(correctPath, {color: "green", interactive: false, distance: json.CurrentPosDistance});
                cppl.id = "correctPath";
                //cppl.addTo(mymap);

                var wppls = [];
                for(i =0; i< json.DistractorEdges.length; i++){

                    var wrongPath = [];

                    for(j=0; j< json.DistractorEdges[i].length; j++){
                        var edge =json.DistractorEdges[i][j].C;
                        wrongPath.push([[edge[0], edge[1]], [edge[2], edge[3]]]);


                    }
                    var wppl = L.polyline(wrongPath, {color:"red",interactive: false, distance: json.DistractorEdgesDistance[i]});
                    wppl.id = "dE"+i;
                    //wppl.addTo(mymap);
                    wppls.push(wppl);
                }



                if(checkQuestions(json.Question)){
                    var q = pickQuestion(json.Question);
                    q.cppl = cppl;
                    q.wppls = wppls;
                    q.img = json.Img;
                    q.abstract = json.Abstract;
                    setQuestion(q);
                }else{
                    console.log("no valid question could be generated :(");
                }


                if(!gameStarted){
                    distanceScore = json.DistanceToTarget;
                    document.getElementById("startWindowContainer").style.visibility = "hidden";
                    document.getElementById("streetSign").style.visibility = "visible";
                    document.getElementById("distanceDisplay").style.visibility = "visible";
                    //document.getElementById("start").style.visibility="hidden";

                    gameStarted = true;
                }

                document.getElementById("streetSign").innerHTML = json.CurrentPos[0].N;
                document.getElementById("distanceDisplay").innerHTML = json.DistanceToTarget+"m to Finish."

                //document.getElementById("answerWindowContainer").style.visibility = "hidden";
                document.getElementById("questionWindowContainer").style.visibility = "visible";
                //document.getElementById("depiction").style.visibility = "hidden";
                mymap.invalidateSize()
    
                console.log("finished calculating shortest path...")
            }

            //unlock map
            document.getElementById("mapid").setAttribute("class", "unlocked");
            deactivateLoader();
            resolve(1)
        }
        }

        var data = JSON.stringify({"do": "quiz"});
    
        console.log("sending json request to server: "+data)
        xhr.send(data);

      });


}







function calcRoute(){
    activateLoader();
    console.log("map cleaned...")


    let promise = new Promise((resolve, reject) => {

        document.getElementById("mapid").setAttribute("class", "locked");

        var xhr = new XMLHttpRequest();
        var url = "/dijkstra";
        xhr.open("POST", url, true);
        xhr.setRequestHeader("Content-Type", "application/json");
    
        xhr.onreadystatechange = function () {
        if (xhr.readyState === 4 && xhr.status === 200) {
            

            germanyLayer = null;
            if(!graphHidden){
                hideGraph();
            }
        
            mymap.eachLayer(function (layer) {
                if(layer.id && layer.id.includes("path")){
                    mymap.removeLayer(layer);
                }
            });

            var json = JSON.parse(xhr.responseText);


            if(json.Distance == 0){
                document.getElementById("routeinfo").innerHTML = "Ops! Couldn't find way?! Check if the markers are placed in a lake, shopping centre or military base :)";

            }else{
                var pl = []
                for (i = 0; i < json.Data.length; i++) {
                    var edge = json.Data[i].C
                    pl.push([[edge[0], edge[1]], [edge[2], edge[3]]])
                }
    
                var polyline = L.polyline(pl, {color: "red", interactive: false})
    
                polyline.id = "path"
                polyline.addTo(mymap);
    
                mymap.invalidateSize()
    
                //update textbox
                if(metric == "distance"){
    
                    if(json.Distance > 1000){
                        var km = (json.Distance/1000).toFixed(2);
                        var t = km+"km"
                    }else{
                        var t= json.Distance+"m"
                    }
                    var text = "Found path for "+mode+" with a distance of "+t;
                }
    
                if(metric == "time"){
    
                    if(mode!= "car"){
                        if(mode=="bike"){
                            speed = 15;
                        }else if(mode=="pedestrian"){
                            speed = 5;
                        }
                        time = 3600 / ((speed*1000) / json.Distance);
                    }else{
                        time = json.Distance;
                    }
    
    
                        var t = time+" seconds"
                        if(time > 60){
                            // give minutes
                            var t = Math.floor(time/60)+" minutes and "+Math.floor(time%60)+" seconds"
                        }
        
                        if (time > 3600){
                            // give hours
                            var t = Math.floor(time/3600)+" hours and "+Math.floor(time%60)+" minutes"
                        }
    
                    var text = "Found path for "+mode+" with a traveltime of "+t;
                }
    
                document.getElementById("routeinfo").innerHTML = text;
    
    
    
    
                console.log("finished calculating shortest path...")
            }

           

            //unlock map
            document.getElementById("mapid").setAttribute("class", "unlocked");
            deactivateLoader();
            resolve(1)
        }
        }

    
    
        //var data = JSON.stringify({"do": "dijkstra", "mode": mode,
          //                          "startLat": marker1.getLatLng().lat, "startLon": marker1.getLatLng().lng,
            //                        "targetLat": marker2.getLatLng().lat, "targetLon": marker2.getLatLng().lng});

        var data = JSON.stringify({"do": "dijkstra", "mode": mode, "metric": metric});
    
        console.log("sending json request to server: "+data)
        xhr.send(data);

      });


}

function selectPede(){
    mode = "pedestrian"
    document.getElementById("car").setAttribute("class", "deactivatedBtn");
    document.getElementById("bike").setAttribute("class", "deactivatedBtn");
    document.getElementById("pedestrian").setAttribute("class", "activatedBtn");

}

function selectBike(){
    mode = "bike"
    document.getElementById("car").setAttribute("class", "deactivatedBtn");
    document.getElementById("bike").setAttribute("class", "activatedBtn");
    document.getElementById("pedestrian").setAttribute("class", "deactivatedBtn");

}


function selectCar(){
    mode = "car"
    document.getElementById("car").setAttribute("class", "activatedBtn");
    document.getElementById("bike").setAttribute("class", "deactivatedBtn");
    document.getElementById("pedestrian").setAttribute("class", "deactivatedBtn");

}

function selectDistance(){
    metric = "distance"
    document.getElementById("time").setAttribute("class", "deactivatedBtn");
    document.getElementById("distance").setAttribute("class", "activatedBtn");
}

function selectTime(){
    metric = "time"
    document.getElementById("time").setAttribute("class", "activatedBtn");
    document.getElementById("distance").setAttribute("class", "deactivatedBtn");
}


//function toggleMapUpdate(){
//    if(autoUpdate){
//        document.getElementById("autoupdate").setAttribute("class", "deactivatedBtn");
//        autoUpdate = false;
//    }else{
//        document.getElementById("autoupdate").setAttribute("class", "activatedBtn");
//        updateMap(mymap, mymap.getBounds(), mymap.getZoom())
//        autoUpdate = true;
//    }
//}


function activateLoader(){
    document.getElementById("loader").style.visibility = "visible";
}

function deactivateLoader(){
    document.getElementById("loader").style.visibility = "hidden";
}

function hideGraph(){

if(!graphHidden){
    mymap.eachLayer(function (layer) {
        if(layer.id && (layer.id.includes("detailZoom") || layer.id.includes("germany"))){
            mymap.removeLayer(layer);
        }
    });
    document.getElementById("hideGraph").innerHTML = "Show graph";
    //document.getElementById("hideGraph").style.background='#17fc03';
    graphHidden = true

}else{

    if(germanyLayer){
        germanyLayer.addTo(mymap);
    }else{
        initMap(mymap)
    }
    
    document.getElementById("hideGraph").innerHTML = "Hide graph";
    //document.getElementById("hideGraph").style.background='#4f4f4f';
    graphHidden = true;
}
}




function geoDistance(lat1, lon1, lat2, lon2) {
	if ((lat1 == lat2) && (lon1 == lon2)) {
		return 0;
	}
	else {
		var radlat1 = Math.PI * lat1/180;
		var radlat2 = Math.PI * lat2/180;
		var theta = lon1-lon2;
		var radtheta = Math.PI * theta/180;
		var dist = Math.sin(radlat1) * Math.sin(radlat2) + Math.cos(radlat1) * Math.cos(radlat2) * Math.cos(radtheta);
		if (dist > 1) {
			dist = 1;
		}
		dist = Math.acos(dist);
		dist = dist * 180/Math.PI;
		dist = dist * 60 * 1.1515;
		return dist = dist * 1.609344;
	}
}

// click handler functions

    // click handler functions
    
    
    
    
    
