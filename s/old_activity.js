goog.provide('nb.act');

/* 
 *  An "activities" page a.k.a. the page where the team looks at a few
 *  activities/puzzles.
 */

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
nb.act.fillUICB = function(e) {
	var obj = this.getResponseJson();
	if (!obj) { return; }
	nb.act.fillUI(obj);
};

nb.act.fillUI = function(obj) {
	if (!obj) { return; }
	if (obj['feedback']) {
		goog.dom.getElement('feedback').innerHTML = obj['feedback'];
	}
	if (obj['hints']) {
		var hl = goog.dom.getElement('hintlist');
		hl.innerHTML = '';
		goog.array.forEach(obj['hints'], function(el, ix, ar) {
				var li = goog.dom.createDom('li');
				li.innerHTML = el;
				goog.dom.appendChild(hl, li);
			})
	}
	if (obj['gossip']) {
		var gl = goog.dom.getElement('gossiplist');
		gl.innerHTML = '';
		var prevDateTime = 0;
		goog.array.forEach(obj['gossip'], function(el, ix, ar) {
                  var li = goog.dom.createDom('li');
		  var d = new Date(el['T']);
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
	}
}

nb.act.guessTimeout = function(e) {
	goog.dom.getElement('feedback').innerHTML = 'Server did not respond. Sorry, the computer is being difficult.';
};

nb.act.onGuessButtonClick = function(e) {
	var xhr = new goog.net.XhrIo();
	var qd = goog.Uri.QueryData.createFromMap({
			'act': goog.global['nickname'],
			'guess': goog.dom.getElement('guess').value,
			'token': goog.global['guesstoken']
		});
	xhr.setTimeoutInterval(15000);
	goog.events.listen(xhr, goog.net.EventType.COMPLETE, nb.act.fillUICB);
	goog.events.listen(xhr, goog.net.EventType.TIMEOUT, nb.act.guessTimeout);
	// domain is octothorpean or localhost
	var url = 'http://www.octothorpean.org/guess';
	if (document.URL.indexOf('localhost') > 1) {
		url = 'http://localhost:8080/guess';
	}
	xhr.send(url, 'POST', qd.toString());
	return false;
};

nb.act.onGuessEnter = function(e) {
	if (e.keyCode == 13) {
		return nb.act.onGuessButtonClick(e);
	}
};

nb.act.onHintButtonClick = function(e) {
	var already = goog.dom.getElement('hintlist')['children'].length;
	var xhr = new goog.net.XhrIo();
	var qd = goog.Uri.QueryData.createFromMap({
			'act': goog.global['nickname'],
			'num': already+1,
			'token': goog.global['hinttoken']
		});
	goog.events.listen(xhr, goog.net.EventType.COMPLETE, nb.act.fillUICB);
	// domain is octothorpean or localhost
	var url = 'http://www.octothorpean.org/hint';
	if (document.URL.indexOf('localhost') > 1) {
		url = 'http://localhost:8080/hint';
	}
	xhr.send(url, 'POST', qd.toString());
	return false;
};

// If the gossip window is open, refresh it
nb.act.maybeSeekGossip = function() {
	if (goog.dom.getElement('gossipbox').style.display != 'none') {
		nb.act.seekGossip();
	}
}
nb.act.seekGossip = function() {
	var xhr = new goog.net.XhrIo();
	goog.events.listen(xhr, goog.net.EventType.COMPLETE, nb.act.fillUICB);
	// domain is octothorpean or localhost
	var url = 'http://www.octothorpean.org/gossip';
	if (document.URL.indexOf('localhost') > 1) {
		url = 'http://localhost:8080/gossip'
	}
	var qd =  goog.Uri.QueryData.createFromMap({
		'act': goog.global['nickname'],
	});
	xhr.send(url, 'POST', qd.toString());
};

nb.act.togglebox = function(id) {
	var togglee = goog.dom.getElement(id);
	if (!togglee) { return; }
	if (togglee.style.display != 'none') {
		togglee.style.display = 'none';
		return;
	}
	goog.dom.getElement('gossipbox').style.display = 'none';
	goog.dom.getElement('hintbox').style.display = 'none';
	goog.dom.getElement('guessbox').style.display = 'none';
	goog.dom.getElement(id).style.display = 'block';
	if (id == 'guessbox') {
		goog.dom.getElement('guess').focus();
	}
};

nb.act.loadActTimeout = function(e) {
	goog.dom.getElement('feedback').innerHTML = 'Server did not respond. Sorry, the computer is being difficult.';
};

nb.act.loadActCB = function(e) {
	var obj = this.getResponseJson();
	if (!obj) return;
	if (obj['act']) {
		goog.global['nickname'] = obj['act'];
		goog.global['guesstoken'] = obj['g'];
		goog.global['hinttoken'] = obj['h'];
		goog.dom.getElement('hintlist').innerHTML = '';
		nb.act.onHintButtonClick();
	}
	nb.act.fillUI(obj);
}

nb.act.loadAct = function(act) {
	var xhr = new goog.net.XhrIo();
	var qd = goog.Uri.QueryData.createFromMap({
			'act': act
		});
	xhr.setTimeoutInterval(15000);
	goog.events.listen(xhr, goog.net.EventType.COMPLETE, nb.act.loadActCB);
	goog.events.listen(xhr, goog.net.EventType.TIMEOUT, nb.act.loadActTimeout);
	var url = 'http://www.octothorpean.org/atokens';
	if (document.URL.indexOf('localhost') > 1) {
		url = 'http://localhost:8080/atokens';
	}
	xhr.send(url, 'POST', qd.toString());
	var uistrip = goog.dom.getElement('uistrip');
	if (uistrip) {
		uistrip.style.display = 'block';
	}
};

nb.act.launchWindow = function(url) {
  var w = window.open(url, '_blank', 
     'height=480,width=640,toolbar=yes,location=yes'); 
  if (window.focus) { 
    w.focus(); 
  }
  return false;
}

nb.act.onLoad = function(e) {
	var gb = goog.dom.getElement('guessbutton');
	if (gb) {
		goog.events.listen(gb, 'click', nb.act.onGuessButtonClick);
		gb.href = '#'; // TODO better way to hover?
	}
	var g = goog.dom.getElement('guess');
	if (g) {
		goog.events.listen(g, 'keyup', nb.act.onGuessEnter);
	}
	var hb = goog.dom.getElement('hintbutton');
	if (hb) {
		goog.events.listen(hb, 'click', nb.act.onHintButtonClick);
		hb.href = '#'; // TODO better way to hover?
	}
	var sgo = goog.dom.getElement('showgossip');
	if (sgo) {
		goog.events.listen(sgo, 'click', goog.partial(nb.act.togglebox, 'gossipbox'));
	}
	var cgo = goog.dom.getElement('closegossip');
	if (cgo) {
		goog.events.listen(cgo, 'click', goog.partial(nb.act.togglebox, 'gossipbox'));
	}
	var shi = goog.dom.getElement('showhints');
	if (shi) {
		goog.events.listen(shi, 'click', goog.partial(nb.act.togglebox, 'hintbox'));
	}
	var chi = goog.dom.getElement('closehints');
	if (chi) {
		goog.events.listen(chi, 'click', goog.partial(nb.act.togglebox, 'hintbox'));
	}
	var sgu = goog.dom.getElement('showguess');
	if (sgu) {
		goog.events.listen(sgu, 'click', goog.partial(nb.act.togglebox, 'guessbox'));
	}
	var cgu = goog.dom.getElement('closeguess');
	if (cgu) {
		goog.events.listen(cgu, 'click', goog.partial(nb.act.togglebox, 'guessbox'));
	}
	nb.act.fillUI(goog.global['initJSON']);
	nb.act.seekGossip();
	var timer = new goog.Timer(1000 * 60 * (0.7 + Math.random()));
	timer.start()
	goog.events.listen(timer, goog.Timer.TICK, nb.act.maybeSeekGossip);
};

goog.events.listen(window, 'load', nb.act.onLoad);
goog.exportSymbol('launchWindow', nb.act.launchWindow);
goog.exportSymbol('loadAct', nb.act.loadAct);
