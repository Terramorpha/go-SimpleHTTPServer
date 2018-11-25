String.prototype.format = function() {
    var formatted = this;
    for (var i = 0; i < arguments.length; i++) {
        var regexp = new RegExp('\\{'+i+'\\}', 'gi');
        formatted = formatted.replace(regexp, arguments[i]);
    }
    return formatted;
};

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
            var v = "value=\"{0}\"".format(settings[key].Value)
            var n = settings[key].Name
            var i = ""
            switch (settings[key].Type) {
                case "int":
                    i += "<input name={0} type=\"number\">".format(key)
                break
                case "bool":
                    var v1 = ""
                    var v2 = ""
                    t = "radio"
                    v1 += "value=\"true\""
                    v2 += "value=\"false\""
                    switch (settings[key].Value){
                        case "true":
                        v1 += " checked=\"checked\""
                        break
                        case "false":
                        v2 += " checked=\"checked\""
                        break
                    }
                        
                    i += "</br><input name=\"{0}\" type=\"{1}\" {2}><label>true</label></br><input name=\"{0}\" type=\"{1}\" {3}><label>false</label>".format(key, t, v1, v2)
                break;
                default:
                i += "<input name=\"{0}\" type=\"${1}\" ${2}></input>".format(key, t, v)
                break
            }
            settingsDiv.innerHTML += "<form method=\"POST\">{0}{1}</br><input type=\"submit\" value=\"update\"></form>".format(key, i)
        })
    }
}





xmlHttp.send(null)
