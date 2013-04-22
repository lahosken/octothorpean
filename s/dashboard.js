var alreadyTime = 0;

var fillUI = function(obj) {
	if (!obj) { return; }
	if (obj['gossip']) {
   	    var ar = []
		var prevDateTime = 0;
        var d = false;
		for (ix = 0; ix < obj['gossip'].length; ix++) {
			el = obj['gossip'][ix]
			d = new Date(el['T']);
			if (d <= alreadyTime) { continue; }
            var tm = '' + d.getMinutes()
			if (tm.length < 2) { tm = '0' + tm; }
            var t = '' + d.getHours() + ':' + tm;
			if (el['T'] < prevDateTime + (60 * 60 * 12)) {
				t = '' + (d.getMonth()+1) + '/' + d.getDate() + ' ' + t;
			}
			ar.push('<li><small>' + t + '</small> ' + el['M'])
		}
		$('#gossip').prepend('<ul>' + ar.join('\n') + '</ul>')
		if (d) {
			alreadyTime = new Date(obj['gossip'][0]['T'])
		}
	}
}

var seekGossip = function() {
	// domain is octothorpean or localhost
	var url = 'http://www.octothorpean.org/gossip';
	if (document.URL.indexOf('localhost') > 1) {
		url = 'http://localhost:8080/gossip';
	}
	if (document['gossipurl']) { url = document['gossipurl']; }
	jQuery.ajax(url, {
		'dataType': 'json',
		'success': fillUI,
		'type': 'POST',		
	})
}

seekGossip();
window.setInterval(seekGossip, 1000 * 10 * (9.9 + Math.random()))