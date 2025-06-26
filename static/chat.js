window.onload = function() {
	const token = localStorage.getItem('token');
	if (!token) {
		location.href = 'login.html';
		return;
	}
	var room = new URLSearchParams(location.search).get('room');
	var room_name = new URLSearchParams(location.search).get('name')
	room_name = String(room_name).charAt(0).toUpperCase() + String(room_name).slice(1);
	document.getElementById('title').innerText = `Room: ${room_name}`;
	document.getElementById('room-name').innerText = room_name;
	var conn;
	var msg = document.getElementById("msg");
	var log = document.getElementById("log");

	function appendLog(item) {
		var doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
		log.appendChild(item);
		if (doScroll) {
			log.scrollTop = log.scrollHeight - log.clientHeight;
		}
	}

	document.getElementById("form").onsubmit = function() {
		if (!conn) { return false; }
		if (!msg.value) { return false; }
		conn.send(JSON.stringify({ type: "message.send", text: msg.value.trim() }));
		msg.value = "";
		return false;
	};

	if (window["WebSocket"]) {
		conn = new WebSocket("ws://" + document.location.host + `/ws/${room}?token=${token}`);
		conn.onclose = function(evt) {
			var item = document.createElement("div");
			item.innerHTML = "<b>Connection closed.</b>";
			appendLog(item);
		};
		conn.onmessage = function(evt) {
			evt.data.split('\n').forEach(line => {
				const msg = JSON.parse(line);
				const date = new Date(msg.data.sent_at);
				const message = date.toLocaleTimeString() + " " + msg.data.username + ": " + msg.data.body;
				var messages = message.split('\n');
				for (var i = 0; i < messages.length; i++) {
					var item = document.createElement("div");
					item.innerText = messages[i];
					appendLog(item);
				}
			});
		};
	} else {
		var item = document.createElement("div");
		item.innerHTML = "<b>Your browser does not support WebSockets.</b>";
		appendLog(item);
	}
};

