var showLert = function(s) {
	$('#feedback').prepend('<div class="lert alert alert-info" style="display:none;">' + s + '<button type="button" class="close" data-dismiss="alert">&times;</button></div>')
    $('.lert').alert().show(0.5)
}

var blork = []

var fillUI = function(obj) {
	if (!obj) { return; }
	if (obj['feedback']) {
		showLert(obj['feedback'])
    }
	if (obj['hints']) {
		$("#hintlist").empty();
        for (ix = 0; ix < obj['hints'].length; ix++) {
			$("#hintlist").append('<li>' + obj['hints'][ix])
		}
	}			
	if (obj['gossip']) {
		$("#gossiplist").empty();
		var prevDateTime = 0;
        for (ix = 0; ix < obj['gossip'].length; ix++) {
            var el = obj['gossip'][ix]
			var d = new Date(el['T']);
			var t = '' + d['getHours']() + ':' + d['getMinutes']();
			var dt = '' + (d['getMonth']()+1) + '/' + d['getDate']() + ' ' + t;
			if (el['T'] < prevDateTime + (60 * 60 * 12)) {
                li = '<li><small>' + t + '</small> ' + el['M'];
            } else {
                prevDateTime = el['T'];
                li = '<li><small>' + dt + '</small> ' + el['M'];
            }
			$("#gossiplist").append(li)
		}
	}
}

var sendGuess = function() {
	// domain is octothorpean or localhost
    var url = 'http://www.octothorpean.org/guess';
    if (document.URL.indexOf('localhost') > 1) {
        url = 'http://localhost:8080/guess';
    }
	jQuery.ajax(url, {
		'data': {
			'act': nickname,
			'guess': $('#guess')[0].value,
			'token': guesstoken,
		},
		'dataType': 'json',
		'error': function(jqxhr, status, error) {
			showLert('Something went wrong talking to server. Status: ' + status + ' Error: ' + error)
		},
		'success': fillUI,
		'timeout': 15000,
		'type': 'POST',
	})
	return false;
}

var seekGossip = function() {
	var url = 'http://www.octothorpean.org/gossip';
	if (document.URL.indexOf('localhost') > 1) {
		url = 'http://localhost:8080/gossip'
	}
	jQuery.ajax(url, {
		'data': {
			'act': nickname,
		},
		'dataType': 'json',
		'success': fillUI,
		'type': 'POST',
	})
}

var seekHint = function() {
	var already = $('#hintlist').children().length;
	var url = 'http://www.octothorpean.org/hint';
	if (document.URL.indexOf('localhost') > 1) {
		url = 'http://localhost:8080/hint';
	}
	jQuery.ajax(url, {
		'data': {
			'act': nickname,
			'num': already+1,
			'token': hinttoken,
		},
		'dataType': 'json',
		'success': fillUI,
		'type': 'POST',		
	})
	return false
}

$('#guessbutton').button().click(sendGuess);
$('#showgossip').click(seekGossip);
$('#hintbutton').click(seekHint);
$('#guess').keydown(function(e){
	if (e.keyCode && e.keyCode == 13) {
		return sendGuess();
	}
});
seekGossip();
fillUI(initJSON);