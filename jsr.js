var container = document.GetElementById("settingsBox")


var req = new XMLHttpRequest()

document.host
req.open("GET", "http://" + document.location.host + "/settings.json")
