function websocket(config) {
	var elstat = config.elementStatus;
	var elinfo = config.elementInfo;
	var reconnectDelay = 1000;
	if ('reconnectDelay' in config) {
		reconnectDelay = config.reconnectDelay;
	}
	var errorReportDelay = 5000;
	if ('errorReportDelay' in config) {
		errorReportDelay = config.errorReportDelay;
	}
	var connected = false;
	var conn;
	function connect() {
		conn = new WebSocket(config.url);
		conn.onopen = function(evt) {
			console.log('websocket connected');
			connected = true;
			elstat.innerHTML = "";
			removeClass(elinfo, "xconn");
		}
		conn.onclose = function(evt) {
			console.log('websocket disconnected');
			connected = false;
			setTimeout(function(){
				connect();
			}, reconnectDelay);
			setTimeout(function(){
				if (!connected) {
					elstat.innerHTML = config.msgClose;
					addClass(elinfo, "xconn");
				}
			}, errorReportDelay);
		}
		conn.onmessage = function(evt) {
			console.log('websocket updated');
			elinfo.innerHTML = evt.data;
		}
	}
	connect();
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
