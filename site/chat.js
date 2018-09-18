//make connection

var socket = io.connect('http://localhost:3000');

//query dom

var message = document.getElementById('message');
    handle = document.getElementById('handle');
    btn = document.getElementById('send');
    output = document.getElementById('output');

//emit events
btn.addEventListener('click', function(){
    //if ((message.value === '') || (handle.value === '')) {
    if (message.value == '') {
    } else {
        socket.emit('chat', {
            message: message.value,
            handle: handle.value
        });
    }
    
    message.value = ""
})

socket.on('chat', function(data){
    output.innerHTML = data.handle + data.message + output.innerHTML});
