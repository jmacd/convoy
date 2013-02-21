var scrapeToken = "Scraper-Token"
var scrapeAction = "Scraper-Action"
var responseUri = "/response"

function respond(token) {
    //console.log("Responding for " + token)
    var xml = new XMLSerializer().serializeToString(document)
    var xhr = new XMLHttpRequest();
    xhr.open("POST", responseUri, true);
    xhr.setRequestHeader(scrapeToken, token)
    xhr.onreadystatechange = function(){ 
	if (xhr.readyState == 4) { 
            if (xhr.status == 200) { 
		var action = xhr.getResponseHeader(scrapeAction)
		//console.log("Finished for " + token + " next " + action)
		if (action == null) {
		    location.reload()
		} else {
		    var newScript = document.createElement('script');
		    newScript.type = 'text/javascript';
		    newScript.innerHTML = action
		    document.body.appendChild(newScript)
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
