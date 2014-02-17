#!/usr/bin/python

import glob
import os.path
import shutil

# TODO statics.go is not the final plan, remember?

MANIFEST = """
cron.yaml
app.yaml
index.yaml
queue.yaml
src/activity.go
src/admin.go
src/badge.go
src/gossip.go
src/octo.go
src/session.go
src/spewmail.go
src/teamui.go
src/teamstore.go
src/templates.go
src/wombat.go
"""

DEST = "srvtest"


def Templates():
  filenames = glob.glob("tmpl/*.html")
  tdotgo = open("src/templates.go", "w")
  tdotgo.write('''package octo

// Don't edit this file. It's automatically generated!

''')
  for filename in filenames:
    lines = open(filename).readlines()
    if lines[0].startswith("#skip"): continue
    madlib = {}
    state = ""
    outs = ""
    for line in lines[1:]:
      line = line.strip()
      if not line.startswith("#"):
        madlib.setdefault(state, "")
        madlib[state] += line + "\n"
        continue
      if not state:
        state = line
        continue
      state = ""
    for line in open("tmpl/"+lines[0].strip()[len("#WRAP:"):-1]).readlines()[1:]:
      line = line.strip()
      if not line.startswith("#"):
        outs += line + "\n"
        continue
      if not line in madlib:
        continue
      outs += madlib[line]
    tdotgo.write("var t%s = `" % filename[5:filename.find(".")])
    tdotgo.write(outs)
    tdotgo.write("`\n\n")
  tdotgo.close()

def Copy(frompath):
  if not frompath: return
  topath = os.path.join(DEST, frompath)
  todir = os.path.dirname(topath)
  if not os.path.isdir(todir):
    os.makedirs(todir)
  shutil.copyfile(frompath, topath)

def Main():
  olds = glob.glob(DEST+"/*")
  for old in olds:
    try:
      shutil.rmtree(old)
    except OSError:
      os.remove(old)
  Templates()
  for line in MANIFEST.split("\n"):
    Copy(line.strip())
  for f in (glob.glob("s/*.html") + glob.glob("s/*.png") + 
            glob.glob("s/*.jpg") + ["s/favicon.ico"] + 
            glob.glob("s/*.css") + glob.glob("s/*.js")):
    Copy(f)
  

Main()
