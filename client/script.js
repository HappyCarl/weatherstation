
var weatherStationAddress = "http://localhost5678"; 

//code from http://www.html5rocks.com/en/tutorials/cors/
function createCORSRequest(method, url) {
  var xhr = new XMLHttpRequest();
  if ("withCredentials" in xhr) {

    // Check if the XMLHttpRequest object has a "withCredentials" property.
    // "withCredentials" only exists on XMLHTTPRequest2 objects.
    xhr.open(method, url, true);

  } else if (typeof XDomainRequest != "undefined") {

    // Otherwise, check if XDomainRequest.
    // XDomainRequest only exists in IE, and is IE's way of making CORS requests.
    xhr = new XDomainRequest();
    xhr.open(method, url);

  } else {

    // Otherwise, CORS is not supported by the browser.
    xhr = null;

  }
  return xhr;
}

var xhr = createCORSRequest('GET', weatherStationAddress);

xhr.onload = function() {
    var responseText = xhr.responseText;
    
    var weatherInfo = JSON.parse(responseText);
    console.log(weatherInfo);
    updateView(weatherInfo);
}
xhr.onerror = function() {
  console.log('There was an error!');
};
if (!xhr) {
  throw new Error('CORS not supported');
} else {
    xhr.send();
}

updateView = function(weatherInfo) {
    temperatureText = document.getElementById("wetter-temperatur");
    humidityText = document.getElementById("wetter-feuchtigkeit");
    windText = document.getElementById("wetter-wind");
    rainText = document.getElementById("wetter-regen");

    temperatureText.innerHTML = weatherInfo.temp + "&deg;"
    humidityText.innerHTML = weatherInfo.humidity + "%"
    windText.innerHTML = weatherInfo.wind_speed + " km/h";
    rainText.innerHTML = "1h: " + weatherInfo.rain.h1 + "mm      24h: " + weatherInfo.rain.h24 + "mm";
    if(weatherInfo.rain.current) {
        document.getElementById("wetter-rain").style.display="inline";
        document.getElementById("wetter-sunny").style.display="none";
    } else {
        document.getElementById("wetter-sunny").style.display="inline";
        document.getElementById("wetter-rain").style.display="none";
    }
}
