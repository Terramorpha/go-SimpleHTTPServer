var req = new XMLHttpRequest();
req.onreadystatechange = function() {
    console.log(req.responseText)
}
xmlHttp.open( "GET", "/settings.json", true ); // false for synchronous request
xmlHttp.send( null );