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

nb.act.categories = [
	'True Name',
	'Place of Death',
	'Date of Death',
	'Color',
	'Animal Aspect',
	'Classification',
	'Favorite Flavor',
]

nb.act.arcrows = []

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
	if (obj['nextacts'] && (obj['nextacts'].length==1)) {
		var whiskTimer = new goog.Timer(1000 * 5);
		whiskTimer.start()
		goog.events.listen(whiskTimer, goog.Timer.TICK, function(ev) {
			document.location = "/a/" + obj['nextacts'][0] + "/"
		})
	}
	goog.global['printfd'].push('alfa')
	if (obj['arcs'] && obj['arcs']['mendy']) {
     	goog.global['printfd'].push('beta')
		var ta = goog.dom.getElement('answersheet');
		var actstates = obj['arcs']['mendy'].ActState
		l = nb.act.categories.length - nb.act.arcrows.length
		for (i = 0; i< l;i++) {
            goog.global['printfd'].push('gamma')
			var tr = goog.dom.createDom('tr');
			nb.act.arcrows.push(tr)
			goog.dom.appendChild(ta, tr);
		}
		for (i = 0; i< nb.act.arcrows.length;i++) {
            goog.global['printfd'].push('delta')
			var tr = nb.act.arcrows[i]
			goog.dom.removeChildren(tr)
			cat = nb.act.categories[i] || "???"
			td = goog.dom.createDom('td', null, cat)
			goog.dom.appendChild(tr, td)
			tex = "???"
			if (actstates[i] && actstates[i].SolvedP) {
				tex = "SOLVED"
			}
			if (actstates[i] && actstates[i].ActID) {
				tex = goog.dom.createDom('a', {href: '/a/' + actstates[i].ActID + '/' }, tex)
			}
			td = goog.dom.createDom('td', null, tex)
			goog.dom.appendChild(tr, td)
		}
	}
   	goog.global['printfd'].push('gamma')
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
	// domain is lahosken0 or localhost
	var url = 'http://lahosken0.appspot.com/guess';
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
	// domain is lahosken0 or localhost
	var url = 'http://lahosken0.appspot.com/hint';
	if (document.URL.indexOf('localhost') > 1) {
		url = 'http://localhost:8080/hint';
	}
	xhr.send(url, 'POST', qd.toString());
	return false;
};

nb.act.seekGossip = function() {
	var xhr = new goog.net.XhrIo();
	goog.events.listen(xhr, goog.net.EventType.COMPLETE, nb.act.fillUICB);
	// domain is lahosken0 or localhost
	var url = 'http://lahosken0.appspot.com/gossip';
	if (document.URL.indexOf('localhost') > 1) {
		url = 'http://localhost:8080/gossip';
	}
	xhr.send(url, 'POST');
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

nb.act.launchWindow = function(url) {
  var w = window.open(url, '_blank', 
     'height=480,width=640,toolbar=yes,location=yes'); 
  if (window.focus) { 
    w.focus(); 
  }
  return false;
}

nb.act.togglebox = function(id) {
	var togglee = goog.dom.getElement(id);
	if (!togglee) { return; }
	if (togglee.style.display == 'none') {
		togglee.style.display = 'block';
	} else {
		togglee.style.display = 'none';
	}
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
	var shi = goog.dom.getElement('showhints');
	if (shi) {
		goog.events.listen(shi, 'click', goog.partial(nb.act.togglebox, 'hintbox'));
	}
	var chi = goog.dom.getElement('closehints');
	if (chi) {
		goog.events.listen(chi, 'click', goog.partial(nb.act.togglebox, 'hintbox'));
	}
	var hb = goog.dom.getElement('hintbutton');
	if (hb) {
		goog.events.listen(hb, 'click', nb.act.onHintButtonClick);
		hb.href = '#'; // TODO better way to hover?
	}
	nb.act.fillUI(goog.global['initJSON']);
};

goog.events.listen(window, 'load', nb.act.onLoad);
goog.exportSymbol('loadAct', nb.act.loadAct);
goog.global['printfd'] = []
