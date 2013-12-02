import collections
import csv
import datetime

INTERESTING_ARCS = [
    "octnbl", "octsbl", "octncy", "octscy",
    "octevi", "octwvi", "octere", "octwre",]

class TLog():
  def __init__(self, created, team, act, verb, guess, num, note):
    self.created = created
    self.team = team
    self.act = act
    self.verb = verb
    self.guess = guess
    self.num = num
    self.note = note

def ReadLogs():
  f = open("../../octodata/dumpteamlogs_20131130.tsv")
  csvr = csv.reader(f, delimiter="\t", lineterminator="\n")
  retval = []
  for tuple in csvr:
    (created_s, team, act, verb, guess, num_s, note) = tuple
    created = datetime.datetime.fromtimestamp(int(created_s, 10))
    num = 0
    if num_s: num = int(num_s, 10)
    retval.append(TLog(created, team, act, verb, guess, num, note))
  return retval

def IndexLogs(l):
  retval = collections.defaultdict(type(dict()))
  for log in l:
    if not log.verb == "solve": continue
    retval[log.team][log.act] = log
  return retval
    
def GetArcs():
  f = open("../../octodata/INTERA_20131130.txt")
  retval = {}
  curnick = ""
  curstruct = {}
  for line in f:
    if line.startswith("NICK"):
      if curnick:
        retval[curnick] = curstruct
      curnick = line[4:].strip()
      curstruct = {}
    if len(line) > 2 and line[1] == ' ':
      if not 'acts' in curstruct: curstruct['acts'] = []
      curstruct['acts'].append(line[2:].strip())
  return retval

arcs = GetArcs()
logs = ReadLogs()
index = IndexLogs(logs)

for arcname in INTERESTING_ARCS:
  arc = arcs[arcname]
  prev_completed_count = 100
  summed_durations = 0
  for puzzle_ix in range(1, len(arc['acts'])-1):
    prev_ix = puzzle_ix - 1
    next_ix = puzzle_ix + 1
    puzzle = arc['acts'][puzzle_ix]
    prev_puzzle = arc['acts'][prev_ix]
    next_puzzle = arc['acts'][next_ix]

    completed_count = 0 
    durations = []

    for team in index:
      ti = index[team]
      if not puzzle in ti: continue
      completed_count += 1
      if not prev_puzzle in ti: continue
      duration = ti[puzzle].created - ti[prev_puzzle].created 
      if duration.total_seconds() < 0: continue
      if next_puzzle in ti and (ti[puzzle].created > ti[next_puzzle].created):
        continue
      durations.append(int(duration.total_seconds()))
    durations.sort()
    twedian_duration = int(durations[len(durations)*2/3]/60)
    notes = []
    percentage = 100 * completed_count / prev_completed_count
    if (puzzle_ix > 1) and percentage < 88:
      notes.append("ARGH")
    if  (puzzle_ix > 1) and twedian_duration > 30:
      notes.append("LONG")
    if puzzle_ix == 1:
      percentage = "-"
      twedian_duration = "-"
    else:
      summed_durations += twedian_duration
    print percentage, completed_count, twedian_duration, puzzle, " ".join(notes)
    prev_completed_count = completed_count
  print summed_durations
  print
  
    
