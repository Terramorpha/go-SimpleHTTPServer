var settingsDiv = document.getElementById("settingsDiv")
console.log("settingsdiv", settingsDiv)
var settings

var xmlHttp = new XMLHttpRequest();
xmlHttp.open("GET", "http://" + location.host + "/settings.json", true)
xmlHttp.onreadystatechange = function() { 
    console.log(xmlHttp.readyState)
    if (xmlHttp.readyState == 4 && xmlHttp.status == 200) {
        console.log(xmlHttp.responseText)
        settings = JSON.parse(xmlHttp.responseText)
        console.log(settings)



        settingsDiv.innerHTML = ""
        Object.keys(settings).sort().forEach(function(key){
            console.log(settings[key])
            var t
            var v = `value="${settings[key].Value}"`
            switch (settings[key].Type) {
                case "bool":
                t = "checkbox"
                if (settings[key].Value == "true") {
                    v = `checked="checked"`

                }
                break;

            }
            settingsDiv.innerHTML += `${key} <input type="${t}" ${v}></input></br>`
        })
    }
}





xmlHttp.send(null)
