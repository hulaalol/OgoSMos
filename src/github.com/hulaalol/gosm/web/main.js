


var markerSet = false;
var marker1;
var marker2;

//var autoUpdate = false;
var mode;
var metric;
var mymap;


var germanyLayer;
var graphHidden;

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


    London = [51.505, -0.09]
    Stuttgart = [48.775, 9.1829321]

    mymap = L.map('mapid',{renderer: L.canvas()}).setView(Stuttgart, 15);


    selectCar();
    selectDistance();
    

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
        maxZoom: 18,
        id: 'mapbox.streets',
        accessToken: 'pk.eyJ1IjoiaHVsYWFsb2wiLCJhIjoiY2szYjBqc2Q3MGhuazNkbXhrbnZsaHIyYiJ9._R-lsR0wjcmJGXf5TRmSSw'
    }).addTo(mymap);


    initMap()

    







});


function sendMarker(marker,type){
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
    graphHidden = false;
}
}

// click handler functions

    // click handler functions
    
    
    
    
    
