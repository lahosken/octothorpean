#!/usr/bin/env python3
import cgi
import cgitb
import json
cgitb.enable()

def levenshteinDistanceOK(s1, s2):
    if len(s1) > len(s2):
        s1, s2 = s2, s1
    tolerance = int(len(s1)/4) + 1
    if len(s2) - len(s1) > tolerance: return False

    distances = range(len(s1) + 1)
    for i2, c2 in enumerate(s2):
        distances_ = [i2+1]
        for i1, c1 in enumerate(s1):
            if c1 == c2:
                distances_.append(distances[i1])
            else:
                distances_.append(1 + min((distances[i1], distances[i1 + 1], distances_[-1])))
        distances = distances_
    return distances[-1] <= tolerance

def canon(s):
    return ''.join([c for c in s if c.isalpha() or c.isdigit()]).upper()

def main():
  PUZS = {'DATA':True}
  print('Content-Type: application/json')
  print()  
  vars = cgi.FieldStorage()
  if not 'puz' in vars:
    print(json.dumps({
      'msg': 'Forgot to specify which puzzle?',
    }))
    return
  if not vars['puz'].value in PUZS:
    print(json.dumps({
      'msg': 'I don\'t know that puzzle ({})!?'.format(vars['puz'].value),
    }))
    return
  puz = PUZS[vars['puz'].value]
  if not 'guess' in vars:
    print(json.dumps({
      'msg': 'Forgot to enter a guess?',
    }))
    return
  if len(vars['guess'].value) <= 0:
    print(json.dumps({
      'msg': 'Forgot to type something?',
    }))
    return
  guess = canon(vars['guess'].value)
  if len(guess) <= 0:
    print(json.dumps({
      'msg': 'Puzzle solutions have letters and numbers.',
    }))
    return
  if guess in puz['soln']:
    unlocks = {}
    for nick in puz['unlocks']:
        unlocks[PUZS[nick]['nickHash']] = nick
    print(json.dumps({
        'msg': 'Yes, {} is the solution.'.format(puz['soln'][0]),
        'soln': puz['soln'][0],
        'unlocks': unlocks,
    }))
    return
  if guess in puz['partials']:
      msg = 'That looks like something you might see "on the way" to the solution. Keep going!'
      if type(puz['partials'][guess]) == type('string'):
          msg = puz['partials'][guess]
      print(json.dumps({
          'msg': msg,
      }))
      return
  for s in puz['soln']:
      if levenshteinDistanceOK(guess, s):
          print(json.dumps({
              'msg': 'That looks close to a solution. Check for typos?',
          }))
          return
  for p in puz['partials']:
      if levenshteinDistanceOK(guess, p):
          print(json.dumps({
              'msg': 'Check for typos?',
          }))
          return
  print(json.dumps({
      'msg': 'Alas, {} is not the answer.'.format(guess),
  }))
          
          

main()
