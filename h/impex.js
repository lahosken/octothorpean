var unlocked = JSON.parse(localStorage.getItem('unlocked') || '{}');
var puzs = JSON.parse(localStorage.getItem('puzs') || '{}')

function persist() {
    localStorage.setItem('unlocked', JSON.stringify(unlocked));
    localStorage.setItem('puzs', JSON.stringify(puzs));
}

function genExprtString() {
    json = {}
    for (var k in unlocked) {
	var nick = unlocked[k];
	json[nick] = {
	    nick: nick,
	    locked: k,
	}
    }
    for (var k in puzs) {
	json[k] = puzs[k];
    }
    return JSON.stringify(json, null, 2);
}

function parseImprtString(s) {
    json = JSON.parse(s);
    var dirtyCount = 0;
    for (var k in json) {
	var dirty = false;
	puz = json[k];
	if (puz.locked && !unlocked[puz.locked]) {
	    unlocked[puz.locked] = k;
	    dirty = true;
	}
	if (!puzs[k]) {
	    puzs[k] = puz;
	    dirty = true;
	}
	for (var kk in puz) {
	    if (!puzs[k][kk]) {
		puzs[k][kk] = puz[kk];
		dirty = true;
	    }
	}
	if (dirty) { dirtyCount++; }
    }
    if (dirtyCount) {
	document.getElementById('importStatus').innerHTML = `Updated ${dirtyCount}, <a href="/">go get 'em</a>&nbsp;#`;
    }
}

function exprt() {
    document.getElementById('exportText').select();
    if (document.execCommand('copy')) {
	document.getElementById('exportStatus').innerHTML = 'Copied to clipboard';
    }
}

async function imprt() {
    const text = await navigator.clipboard.readText();
    parseImprtString(text);
    persist();
}

function onStart() {
    st = document.getElementById('exportText');
    st.value = genExprtString();
    st.setAttribute('readOnly', 'true');
    document.getElementById('exportBtn').addEventListener('click', exprt);
    document.getElementById('importBtn').addEventListener('click', imprt);
}

onStart();
