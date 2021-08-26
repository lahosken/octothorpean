import base64
import chevron
import hashlib
import json
import os
import shutil

ACT_TEMPLATE = 'octoast/Act.html'
ARC_TEMPLATE = 'octoast/Arc.html'
ACT_JS_TEMPLATE = 'octoast/p.js'
ARC_JS_TEMPLATE = 'octoast/r.js'

class Puz:
    '''Info about one puzzle: title, solution, etc'''
    
    def __init__(self):
        self.title = ''
        self.nick = ''
        self.nickHash = ''
        self.solution = []
        self.hints = []
        self.partials = {}
        self.unlocks = []
        self.arcs = []

class Arc:
    '''An "arc" (sequence) of puzzles'''

    def __init__(self):
        self.nick = ''
        self.title = ''
        self.icon = ''
        self.show = False
        self.puzzles = []

def b64e(b):
    return base64.b64encode(b).decode('utf-8')

def hashNick(s):
    m = hashlib.sha256()
    m.update(bytes(s, 'utf-8'))
    return m.hexdigest()[:8]

def canon(s):
    return ''.join([c for c in s if c.isalpha() or c.isdigit()]).upper()

def readPuzTxt(nick, puzs):
    f = open(os.path.join('/home/lahosken/lab/octopuz/', nick, 'puz.txt'))
    for line in f.readlines():
        if line.startswith('TITLE'):
            puzs[nick].title = line[5:].strip()
        if line.startswith('SOLUTION'):
            puzs[nick].solution.append( canon(line[8:].strip()) )
        if line.startswith('PARTIAL'):
            l = line[7:].strip().split()
            puzs[nick].partials[canon(l[0])] = ' '.join(l[1:]) or 1
        if line.startswith('HINT'):
            puzs[nick].hints.append(line[4:].strip())

def readPuzTxts(puzs):
    for nick in puzs:
        readPuzTxt(nick, puzs)

def genArc(arc, puzs):
    nick = arc.nick
    outDirPath = os.path.join('/home/lahosken/octoast/arc/', nick)
    os.makedirs(outDirPath, exist_ok=True)
    html_s = chevron.render(
        open(ARC_TEMPLATE, 'r'), {
            'PageTitle': '{} Arc # Octothorpean '.format(arc.title),
            'NavTitle': 'Arc: {}'.format(arc.title),
            'Guts': '',
        })
    htmlFileName = os.path.join(outDirPath, 'index.html')
    htmlFile = open(htmlFileName, 'w')
    htmlFile.write(html_s)
    htmlFile.close()
    js_s = chevron.render(
        open(ARC_JS_TEMPLATE, 'r'), {
            'Locked': [puzs[p].nickHash for p in arc.puzzles],
        })
    jsFileName = os.path.join(outDirPath, 'r.js')
    jsFile = open(jsFileName, 'w')
    jsFile.write(js_s)
    jsFile.close()
    
def genPuz(nick, puzData, arcs, puzs):
    inDirPath = os.path.join('/home/lahosken/lab/octopuz/', nick)
    indexTextF = open(os.path.join(inDirPath, 'index.html'))
    outDirPath = os.path.join('/home/lahosken/octoast/a/', nick)
    os.makedirs(outDirPath, exist_ok=True)
    shutil.copytree(inDirPath, outDirPath, dirs_exist_ok=True)
    html_s = chevron.render(
        open(ACT_TEMPLATE, 'r'), {
            'PageTitle': '{} # Octothorpean '.format(puzData.title),
            'NavTitle': puzData.title,
            'Guts': indexTextF.read(),
        })
    htmlFileName = os.path.join(outDirPath, 'index.html')
    htmlFile = open(htmlFileName, 'w')
    htmlFile.write(html_s)
    htmlFile.close()
    b64Hints = [b64e(bytes(h, 'utf-8')) for h in puzData.hints]
    shownArcs = {}
    for arc in [arcs[a] for a in puzData.arcs]:
        if not arc.show: continue
        shownArcs[arc.nick] = {
            'nick': arc.nick,
            'title': arc.title,
            'icon': arc.icon,
            'puzzles': [puzs[p].nickHash for p in arc.puzzles],
            }
    js_s = chevron.render(
            open(ACT_JS_TEMPLATE, 'r'), {
            'Nick': nick,
            'NickHash': puzData.nickHash,
            'Title': json.dumps(puzData.title),
            'Hints': json.dumps(b64Hints, indent=2),
            'Arcs': json.dumps(shownArcs, indent=2),
        })
    jsFileName = os.path.join(outDirPath, 'p.js')
    jsFile = open(jsFileName, 'w')
    jsFile.write(js_s)
    jsFile.close()

def readInteraTxt():
    puzs = {}
    arcs = {}
    nick = ''
    prev = ''
    intera_txt_f = open('/home/lahosken/lab/octopuz/INTERA.txt')
    already ={}
    for line in intera_txt_f:
        if line.startswith('NICK'):
            nick = line[4:].strip()
            arcs[nick] = Arc()
            arcs[nick].nick = nick
            prev = ''
            continue

        if line.startswith('TITLE'):
            arcs[nick].title = line[5:].strip()
            continue

        if line.startswith('ICONLINK'):
            arcs[nick].show = True
            continue

        if line.startswith('ICON'):
            arcs[nick].icon = line[4:].strip()
            continue

        if line[0].isalpha() and line[1] == ' ':
            n = line[1:].strip()
            if not n in puzs:
                puzs[n] = Puz()
                puzs[n].nick = n
                puzs[n].nickHash = hashNick(n)
            arcs[nick].puzzles.append(n)
            if arcs[nick].show:
                puzs[n].arcs.append(nick)
            if prev and line[0] != 'X':
                puzs[prev].unlocks.append(n)
            if line[0] in 'ANX':
                prev = n
            continue
    return puzs, arcs

def copyStaticFiles():
    shutil.copytree(
        '/home/lahosken/lab/pexy/octoast/h',
        '/home/lahosken/octoast/h',
        dirs_exist_ok=True)

def genChecker(puzs):
    pj = {}
    for puz in puzs.values():
        pj[puz.nick] = {
            "soln": puz.solution,
            "partials": puz.partials,
            "unlocks": puz.unlocks,
            "nickHash": puz.nickHash,
        }
        
    os.makedirs('/home/lahosken/octoast/cgi-bin', exist_ok=True)
    f = open('/home/lahosken/octoast/cgi-bin/solucheck', 'w')
    f.write(open("octoast/solucheck.py").read().replace('''{'DATA':True}''', json.dumps(pj, indent=2)))
    f.close()
    os.chmod('/home/lahosken/octoast/cgi-bin/solucheck', 33268) # |executable

def main():
    copyStaticFiles()
    puzs, arcs = readInteraTxt()
    readPuzTxts(puzs)
    for nick, puzData in puzs.items():
        genPuz(nick, puzData, arcs, puzs)
    genChecker(puzs)
    for a in arcs.values():
        genArc(a, puzs)
    shutil.copyfile('octoast/Top.html', '/home/lahosken/octoast/index.html')
    shutil.copyfile('octoast/Top.html', '/home/lahosken/octoast/a/index.html')
    shutil.copyfile('octoast/Top.html', '/home/lahosken/octoast/arc/index.html')
    os.makedirs('/home/lahosken/octoast/impex/', exist_ok=True)
    shutil.copyfile('octoast/ImpEx.html', '/home/lahosken/octoast/impex/index.html')

if __name__ == '__main__':
    main()
