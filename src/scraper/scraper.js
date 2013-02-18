var scrapeUri = "/scrape"
var responseUri = "/response"
var scrapeHeader = "Scraper-Token"

console.log('Scraper starting...')

function respond(token, xml) {
    var xhr = new XMLHttpRequest();
    xhr.open("POST", responseUri, true);
    xhr.setRequestHeader(scrapeHeader, token)
    xhr.onreadystatechange = function(){ 
	if (xhr.readyState == 4) { 
            if (xhr.status == 200) { 
		console.log("Finished for " + token)
            } else { 
		console.log("Status is " + xhr.status + " for " + token); 
            }
	    connect();  // Repeat!
	}
    }
    if (xhr.readyState == 0) { 
	console.log("Error: " + xhr.readyState + ' ' + xhr.responseText +
		    " for " + token); 
    } else {
	xhr.send(xml);
    }
}

function connect() {
    var xhr = new XMLHttpRequest();
    xhr.open("GET", scrapeUri, true);
    xhr.onreadystatechange = function(){ 
	if (xhr.readyState == 4) { 
            if (xhr.status == 200) { 
		var evaltext = xhr.responseText; 
		document.body.removeChild(document.body.lastChild);
		var el = document.createElement('div');
		el.innerHTML = evaltext
		document.body.appendChild(el)
		var token = xhr.getResponseHeader(scrapeHeader)
		var xml = new XMLSerializer().serializeToString(document);
		respond(token, xml)
            } else { 
		console.log("Status is " + xhr.status); 
            }
	}
    }
    if (xhr.readyState == 0) { 
	console.log("Error: " + xhr.readyState + ' ' + xhr.responseText); 
    } else {
	xhr.send();
    }
}

connect();
