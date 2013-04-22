goog.provide('nb.dashboard');

goog.require('goog.array');
goog.require('goog.dom');
goog.require('goog.events');
goog.require('goog.net.XhrIo');
goog.require('goog.Timer');
goog.require('goog.Uri.QueryData');

/**
 * callback for most of our requests: We get messages from the server that
 * are for various pieces of UI: we should display hints in the hint list,
 * guess feeedback in the feedback box,...
 */
nb.dashboard.fillUICB = function(e) {
	var obj = this.getResponseJson();
	if (!obj) { return; }
	nb.dashboard.fillUI(obj);
};

nb.dashboard.alreadyTime = new Date(0);

nb.dashboard.fixme = 12;

nb.dashboard.fillUI = function(obj) {
	if (!obj) { return; }
	if (obj['gossip']) {
		var gd = goog.dom.getElement('gossip');
   	    var gl = goog.dom.createDom('ul', {})
		var prevDateTime = 0;
                var d = false;
		goog.array.forEach(obj['gossip'], function(el, ix, ar) {
                  var li = goog.dom.createDom('li');
		  d = new Date(el['T']);
                  if (d <= nb.dashboard.alreadyTime) { return; }
                  var t = '' + d['getHours']() + ':' + d['getMinutes']();
                  var dt = '' + (d['getMonth']()+1) + '/' + d['getDate']() + ' ' + t;
		  if (el['T'] < prevDateTime + (60 * 60 * 12)) {
					li.innerHTML = '<small>' + t + '</small> ' + el['M'];
				} else {
					prevDateTime = el['T'];
					li.innerHTML = '<small>' + dt + '</small> ' + el['M'];
				}
				goog.dom.appendChild(gl, li);
		})
            if (d) {
		nb.dashboard.alreadyTime = new Date(obj['gossip'][0]['T'])
	    }
	    nb.dashboard.fixme = gl;
            if (! gl.childNodes.length) { return; }
	    gd.insertBefore(gl, goog.dom.getFirstElementChild(gd));
	}
}

nb.dashboard.seekGossip = function() {
	var xhr = new goog.net.XhrIo();
	goog.events.listen(xhr, goog.net.EventType.COMPLETE, nb.dashboard.fillUICB);
	// domain is octothorpean or localhost
	var url = 'http://www.octothorpean.org/gossip';
	if (document.URL.indexOf('localhost') > 1) {
		url = 'http://localhost:8080/gossip';
	}
	xhr.send(url, 'POST');
};

nb.dashboard.onLoad = function(e) {
	nb.dashboard.seekGossip();
	var timer = new goog.Timer(1000 * 10 * (4.9 + Math.random()));
	timer.start()
	goog.events.listen(timer, goog.Timer.TICK, nb.dashboard.seekGossip);
}

goog.events.listen(window, 'load', nb.dashboard.onLoad);