var unlocked = JSON.parse(localStorage.getItem('unlocked') || '{}');
var puzs = JSON.parse(localStorage.getItem('puzs') || '{}');

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
	    li.innerHTML = `<a href="/a/${u}">${ti}</a>`;
	} else {
	    li.innerHTML = `<em>???</em>`;
	}
	parent.append(li);
    }
}

function onStart() {
    var parent = document.getElementById('puzList');
    fillArc(locked, parent);
}
onStart();
