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

print "NUM TEAMS:", len(index)

for arcname in INTERESTING_ARCS:
  arc = arcs[arcname]
  # Don't start w/eightways. Most teams work on one or two
  # arcs at a time. If it take a long time to start a puzzle
  # 'unlocked' by eightways, that doesn't mean it's impossible;
  # maybe that team started on something else.
  start_act = arc['acts'][1]
  # Don't end with conspiracy. It's a scary meta. A team might
  # not try to solve it after solving their first arc, e.g.
  end_act = arc['acts'][-2]

  began_count = 0
  ended_count = 0
  durations = []

  for team in index:
    if start_act in index[team]:
      began_count += 1
    if end_act in index[team]:
      ended_count += 1
      if start_act in index[team] and index[team][end_act].created > index[team][start_act].created:
        durations.append(int((index[team][end_act].created -
                          index[team][start_act].created).total_seconds()))
          
  durations.sort()
  median_duration = int(durations[len(durations)/2]/60)
  print arcname, 100 * ended_count/began_count, ended_count, "/", began_count, median_duration
  
