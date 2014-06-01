function websocket(config) {
	var elstat = config.elementStatus;
	var elinfo = config.elementInfo;
	var conn = new WebSocket(config.url);
	var reconnectDelay = 1000;
	if ('reconnectDelay' in config) {
		reconnectDelay = config.reconnectDelay;
	}
	conn.onclose = function(evt) {
		elstat.innerHTML = config.msgClose;
		addClass(elinfo, "xconn");
		window.setTimeout(websocket(config), reconnectDelay);
	}
	conn.onopen = function(evt) {
		elstat.innerHTML = "";
		console.log('websocket connected');
		removeClass(elinfo, "xconn");
	}
	conn.onmessage = function(evt) {
		console.log('websocket updated');
		elinfo.innerHTML = evt.data;
	}
}
function addClass(el, cls) {
	if (el.className == "") {
		el.className = cls;
		return
	}
	var v0 = el.className.split(" ");
	for (var i = 0; i < v0.length; i++) {
		if (v0[i] == cls)
			return;
	}
	v0.push(cls);
	el.className = v0.join(" ");
}
function removeClass(el, cls) {
	if (el.className == "") {
		return;
	}
	if (el.className == cls) {
		el.className = "";
		return;
	}
	var v0 = el.className.split(" ");
	var v1 = [];
	for (var i = 0; i < v0.length; i++) {
		if (v0[i] != "" && v0[i] != cls)
			v1.push(v0[i])
	}
	el.className = v1.join(" ");
}
