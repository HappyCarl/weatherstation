
console.log("Loading weather data...")

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

var xhr = createCORSRequest('GET', "http://ipecho.net/plain");
xhr.onload = function() {
    var responseText = xhr.responseText;
    console.log(responseText);
}
xhr.onerror = function() {
  console.log('There was an error!');
};
if (!xhr) {
  throw new Error('CORS not supported');
} else {
    xhr.send()
}

templateDiv = document.getElementById("wetterstation-wetter")
