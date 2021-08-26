var unlocked = JSON.parse(localStorage.getItem('unlocked') || '{}');
var puzs = JSON.parse(localStorage.getItem('puzs') || '{}')
var hist = JSON.parse(localStorage.getItem('hist') || '[]');

function onStart() {
    if (!unlocked[locked]) {
	unlocked[locked] = nick;
    }

    if (!puzs[nick]) {
	puzs[nick] = {}
    }
    puzs[nick].nick = nick;
    puzs[nick].locked = locked;
    puzs[nick].title = title;

    if (puzs[nick].soln) {
	var parent = document.getElementById('lefty');
	parent.innerHTML = `
          <div class="container bg-white text-info">
             Solved! Answer is: <strong>${puzs[nick].soln}</strong>
          </div>
        `;
    }

    if (Object.keys(arcs).length > 0) {
	var parent = document.getElementById('arcs');
	for (var k in arcs) {
	    var arc = arcs[k];
	    var btn = document.createElement('button');
	    btn.setAttribute('type', 'button');
	    btn.setAttribute('class', 'btn btn-light');
	    var img = document.createElement('img');
	    img.setAttribute('src', arc.icon);
	    img.setAttribute('class', 'arcIcon')
	    btn.append(img);
	    parent.append(btn);
	    btn.arc = arc;
	    var handler = function(ev) {
		var ul = document.getElementById('arcPuzList');
		ul.innerHTML = '';
		document.getElementById('arcModalLabel').innerHTML =
		    'Arc: ' + this.arc.title;
		fillArc(this.arc.puzzles, ul);
		var m = new bootstrap.Modal(document.getElementById('arcModal'), {});
		m.show();
	    };
	    btn.addEventListener('click', handler);
	}
    }

    bspans = document.getElementsByTagName('blanks');
    for (ix = 0; ix < bspans.length; ix++) {
	bspan = bspans[ix];
	var s = bspan.innerHTML;
	s = s.replace(/\./g, '_');
        s = s.replace(/O/g, '&#9711;');
        s = s.replace(/ /g, ' &nbsp; ');
        s = s.replace(/__/g, '_&nbsp;_');
        s = s.replace(/__/g, '_&nbsp;_');
        bspan.innerHTML = s;

    }

    persist();
}
onStart();

function persist() {
    localStorage.setItem('unlocked', JSON.stringify(unlocked));
    localStorage.setItem('puzs', JSON.stringify(puzs));
    localStorage.setItem('hist', JSON.stringify(hist));
}

function sendGuess() {
    var guess = document.getElementById('guess').value;
    var r = new XMLHttpRequest();
    r.open('GET', `/cgi-bin/solucheck?puz=${nick}&guess=${guess}`);
    r.responseType = 'json';
    r.onload = function() {
	t = Date.now();
	j = r.response;
	if (j.msg) {
	    if (hist.length > 50) {
		hist = hist.slice(30);
	    }
	    hist.push({
		t: t,
		msg: j.msg,
		puz: nick,
	    })
	    showMsg(j.msg, t);
	}
	if (j.unlocks  && Object.keys(j.unlocks).length > 0) {
	    var msgHTML = 'Unlocked:';
	    for (const locked in j.unlocks) {
		var u = j.unlocks[locked];
		unlocked[locked] = u;
		msgHTML += ` <a href="/a/${u}/">${u}</a>`;
		hist.push({
		    t: t,
		    msg: 'Unlocked',
		    puz: u,
		});
	    }
	    puzs[nick].unlocks = j.unlocks;
	    hist.push({
		t: t,
		msg: msgHTML,
		puz: nick,
	    })
	    showMsg(msgHTML, t);
	}
	if (j.soln) {
	    puzs[nick].soln = j.soln;
	}
	persist();
    }
    r.send();
}

var eg = document.getElementById('enterGuess');
if (eg) {
    eg.addEventListener('click', sendGuess);
    document.getElementById('guess').addEventListener('keyup', function(ev) {
	if (ev.keyCode == 13) { ev.preventDefault(); sendGuess(); }
    });
}

function fillArc(arcPuzList, parent) {
    for (var ix = 0; ix < arcPuzList.length; ix++) {
	var li = document.createElement('li');
	var lo = arcPuzList[ix];
	if (unlocked[lo]) {
	    var u = unlocked[lo];
	    var ti = u;
	    if (puzs[u] && puzs[u].title) {
		ti = puzs[u].title;
	    }
	    var cl = 'link-primary';
	    var so = '';
	    if (puzs[u] && puzs[u].soln) {
		cl = 'link-secondary';
		so = '<em class="text-secondary">Solved!</em>'
	    }
	    li.innerHTML = `<a href="/a/${u}" class="${cl}">${ti}</a> ${so}`;
	} else {
	    li.innerHTML = `<em>???</em>`;
	}
	parent.append(li);
    }
}

function showMsg(html, timestamp) {
    var parent = document.getElementById('feedback');
    if (parent.childElementCount > 5) {
	parent.firstElementChild.remove();
	parent.firstElementChild.remove();
    }
    var e = document.createElement('div');
    e.setAttribute('class', 'msgBox container');
    var t = document.createElement('div');
    t.setAttribute('class', 'msgTxt');
    t.innerHTML = html;
    e.append(t);
    var tsd = document.createElement('div');
    tsd.setAttribute('class', 'msgDate');
    date = new Date(timestamp)
    var tst = document.createTextNode(date.toLocaleDateString());
    tsd.append(tst);
    e.append(tsd);
    var x = document.createElement('button');
    x.setAttribute('type', 'button');
    x.setAttribute('class', 'btn-close msgCloseButton');
    x.setAttribute('aria-label', 'Dismiss message');
    x.addEventListener('click', function(ev) { e.remove(); });
    e.append(x);
    parent.append(e);
}

document.getElementById('oneMoreHint').addEventListener('click', function(ev) {
    var ul = document.getElementById('hintList');
    var html = atob(hints[ul.childElementCount]);
    var li = document.createElement('li');
    li.innerHTML = html;
    ul.append(li);
    var btn = document.getElementById('oneMoreHint');
    if (ul.childElementCount >= hints.length) {
	btn.disabled = true;
	btn.innerText = 'That was the last hint.'
    } else {
	btn.innerText = 'Show another Hint'
    }
})
