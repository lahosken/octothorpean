import collections
import csv
import datetime

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
  f = open("../../octodata/dumpteamlogs_20140120.tsv")
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
  d = collections.defaultdict(type(0))
  for log in l:
    if not log.verb == "solve": continue
    d[log.act] += 1
  return d
    
logs = ReadLogs()
index = IndexLogs(logs)
for key in index:
  print index[key], key
  
