var unlocked = JSON.parse(localStorage.getItem('unlocked') || '{}');
var puzs = JSON.parse(localStorage.getItem('puzs') || '{}')
var hist = JSON.parse(localStorage.getItem('hist') || '[]');

/*
 * Return HTML string of link to a puzzle.
 */
function pLinkHTML(nick) {
    var ti = nick;
    var cl = 'link-primary';
    if (puzs[nick] && puzs[nick].title)  {
	ti = puzs[nick].title.replace(/ /g, '&nbsp;');
    }
    if (puzs[nick] && puzs[nick].soln)  {
	cl = 'link-secondary';
    }
    return `<a href="/a/${nick}/" class="${cl}">${ti}</a>`;
}

function onStart() {
    already = {}

    var ul = document.createElement('ul');
    for (var ix = hist.length-1; ix >= 0; ix--) {
	h = hist[ix];
	html = h.msg;
	if (h.puz) {
	    if (already[h.puz]) { continue }
	    already[h.puz] = true;
	    html = `${pLinkHTML(h.puz)}: ${h.msg}`;
	}
	var li = document.createElement('li');
	li.innerHTML = html;
	ul.append(li);
	if (ul.childElementCount > 4) { break }
    }
    if (ul.childElementCount > 2) {
	var parent = document.getElementById('histSection');
	var h5 = document.createElement('h5');
	h5.innerHTML = 'Your Recent Work'
	parent.append(h5);
	parent.append(ul);
	p = document.createElement('p');
	p.innerHTML = ('# # #');
	parent.append(p);
    } else {
	already = {};
    }


    var morePuzzles = [];
    for (var key in unlocked) {
	var puz = unlocked[key];
	if (already[puz]) { continue }
	morePuzzles.push(puz);
    }
    if (morePuzzles.length > 2) {
	var parent = document.getElementById('plistSection');
	var h5 = document.createElement('h5');
	h5.innerHTML = 'Puzzles Puzzles Puzzles'
	parent.append(h5);
	var p = document.createElement('p');
	var links = [];
	for (var ix = 0; ix < morePuzzles.length; ix++) {
	    links.push(pLinkHTML(morePuzzles[ix]));
	}
	p.innerHTML = links.join('&nbsp;# ');
	parent.append(p);
	p = document.createElement('p');
	p.innerHTML = ('# # #');
	parent.append(p);
    }
}
onStart();
