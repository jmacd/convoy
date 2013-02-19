var scrapeUri = "/scrape"
var responseUri = "/response"
var scrapeHeader = "Scraper-Token"
var scrapeAction = "Scraper-Action"

console.log('Scraper starting...')

function respond(token, xml) {
    var xhr = new XMLHttpRequest();
    xhr.open("POST", responseUri, true);
    xhr.setRequestHeader(scrapeHeader, token)
    xhr.onreadystatechange = function(){ 
	if (xhr.readyState == 4) { 
            if (xhr.status == 200) { 
		var action = xhr.getResponseHeader(scrapeAction)
		console.log("Finished for " + token + " next " + action)
		if (action == null) {
		    connect();  // Repeat!
		} else {
		    var newScript = document.createElement('script');
		    newScript.type = 'text/javascript';
		    newScript.innerHTML = action
		    document.body.firstChild.appendChild(newScript)

		    var newxml = 
			new XMLSerializer().serializeToString(document);
		    respond(token, newxml)
		}
            } else { 
		console.log("Status is " + xhr.status + " for " + token); 
            }
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
		// // Re-inject the scripts!
		// var scripts = document.getElementsByTagName('script')
		// console.log("Found " + scripts.length + " scripts")
		// for (var i = 0; i < scripts.length; i++) {
		// console.log(scripts[i])
		// var oldchild = scripts[i].parentNode.removeChild(scripts[i])
		// document.body.firstChild.appendChild(oldchild)
		// }		
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
