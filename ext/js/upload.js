successfiles = [];
Dropzone.options.upload = {
	maxFilesize: 2048, // megabytes
	init: function() {
		this.on("addedfile", function(file) {
			uploadFinished(false);
		});
		this.on("queuecomplete", function() {
			while (successfiles.length != 0) {
				this.removeFile(successfiles.pop());
			}
			uploadFinished(true);
		});
		this.on("success", function(file) {
			successfiles.push(file);
		});
	}
};
function uploadFinished(finished) {
	var el = document.getElementById("browser");
	if (finished) {
		removeClass(el, "hidden");
	} else {
		addClass(el, "hidden");
	}
}
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
			elstat.innerHTML = config.msgConnectionActive;
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
					elstat.innerHTML = "";
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
