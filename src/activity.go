package octo

/*
 * Activities (aka Puzzles) and groups of activities 
 * (aka page that lists puzzles a team has "unlocked" w/solving status?...
 *  except that we're still figuring out what this means...)
 */

import (
	"appengine"
	"appengine/datastore"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	//	"log"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

type ActivityRecord struct {
	Nickname  string
	Title     string `datastore:",noindex"`
	GCNote    string `datastore:",noindex"`
	URL       string `datastore:",noindex"`
	Guts      []byte
	Solutions []string `datastore:",noindex"`
	Partials  []string `datastore:",noindex"`
	Hints     []string `datastore:",noindex"`
	Tags      []string `datastore:",noindex"`
	IVars     []string `datastore:",noindex"` // INTERA vars // unused?
}

type ICache struct {
	Icon     string
	Iconlink string
	Next     []string
}

type ArcCache struct {
	Nick  string
	Icon  string
	Title string
	Type  string
	Act   []string
}

var interacache = map[string]ICache{}
var arccache = map[string]ArcCache{}

type OneArc struct {
	Nick     string
	Icon     string
	Title    string
	Furthest string
	ActState []TAStateRecord
}

// return nicely JSONicable map of team's state in some arcs
func arcmaps(context appengine.Context, tid string, arcs []string) map[string]OneArc {
	m := map[string]OneArc{}
	// if anyone ever wants multiple arcs, fetching state records for
	// one arc at a time is kina slow.
	for _, arcname := range arcs {
		arc := fetcharc(context, arcname)
		var keys []*datastore.Key
		for _, actID := range arc.Act {
			keys = append(keys, datastore.NewKey(context, "TAState", actID+":"+tid, 0, nil))
		}
		var states []TAStateRecord = make([]TAStateRecord, len(keys))
		datastore.GetMulti(context, keys, states)
		furthest := ""
		for _, tas := range states {
			if tas.TeamID != "" {
				furthest = tas.ActID
			} else {
				break
			}
		}
		m[arcname] = OneArc{arcname, arc.Icon, arc.Title, furthest, states}
	}
	return m
}

// serve JSON about some arcs
func arcjson(w http.ResponseWriter, r *http.Request) {
	_, tid := GetAndOrUpdateSession(w, r)
	context := appengine.NewContext(r)
	w.Header().Set("Content-Type", "text/javascript")
	arclist := strings.Split(r.FormValue("arcs"), ",")
	js := MapSI{"arcs": arcmaps(context, tid, arclist)}
	spewjsonp(w, r, js)
}

// Show an "arc"; roughly, a sequence of activities
func arc(w http.ResponseWriter, r *http.Request) {
	_, tid := GetAndOrUpdateSession(w, r)
	if tid == "" {
		showMessage(w, "Go back", "These \"arc\" pages aren't much use unless you're logged in--they help you keep track of which activities you've completed.", tid, "")
		return
	}
	context := appengine.NewContext(r)
	urlparts := strings.Split(r.URL.Path, "/")
	if len(urlparts) < 3 {
		return
	}
	arcID := urlparts[2]
	arc := fetcharc(context, arcID)
	if arc.Nick == "" {
		w.WriteHeader(http.StatusNotFound)
		showMessage(w, "No such arc", "No such arc", tid, "")
		return
	}
	var keys []*datastore.Key
	for _, actID := range arc.Act {
		keys = append(keys, datastore.NewKey(context, "TAState", actID+":"+tid, 0, nil))
	}
	var states []TAStateRecord = make([]TAStateRecord, len(keys))
	datastore.GetMulti(context, keys, states)
	t := template.Must(template.New("").Parse(tArc))
	t.Execute(w, MapSI{
		"PageTitle": arc.Title,
		"TID":       tid,
		"Arc":       arc,
		"States":    states,
	})
}

func activity(w http.ResponseWriter, r *http.Request) {
	session, tid := GetAndOrUpdateSession(w, r)
	context := appengine.NewContext(r)
	urlparts := strings.Split(r.URL.Path, "/")
	if len(urlparts) < 3 {
		context.Errorf("ACT strange path too few parts %s", r.URL.Path)
		return
	}
	actID := urlparts[2]
	if actID == "" {
		w.WriteHeader(http.StatusNotFound)
		showMessage(w, "No such activity", "No such activity", tid, "")
		return
	}
	// if URL is something like /a/xwd or /a/xwd/ , 
	// show the xwd puzzle's "index.html" (stored in act's Guts)
	if len(urlparts) == 3 || (len(urlparts) == 4 && urlparts[3] == "") {
		showActPage(w, r, session, tid, context, actID)
		return
	}
	// if URL is like /a/xwd/grid.png, serve file from the ActFS xwd/grid.png
	if len(urlparts) > 3 {
		showActFS(w, r, session, tid, context, actID, urlparts)
		return
	}
}

func activityjson(w http.ResponseWriter, r *http.Request) {
	session, tid := GetAndOrUpdateSession(w, r)
	context := appengine.NewContext(r)
	actID := r.FormValue("act")
	if actID == "" {
		js := MapSI{
			"err": "no act",
			"act": actID,
		}
		spewjsonp(w, r, js)
		return
	}
	key := datastore.NewKey(context, "Activity", actID, 0, nil)
	act := ActivityRecord{}
	err := datastore.Get(context, key, &act)
	if err == datastore.ErrNoSuchEntity {
		js := MapSI{
			"err": "no such act",
			"act": actID,
		}
		spewjsonp(w, r, js)
		return
	}
	if err != nil {
		js := MapSI{
			"err": err.Error(),
			"act": actID,
		}
		spewjsonp(w, r, js)
		return
	}
	tas := TAStateRecord{}
	if tid != "" {
		key = datastore.NewKey(context, "TAState", actID+":"+tid, 0, nil)
		err = datastore.Get(context, key, &tas)
		if err != nil && err != datastore.ErrNoSuchEntity {
			context.Warningf("View TAS hit ERR %s", err.Error())
		}
	}
	if tas.Hints > len(act.Hints) {
		tas.Hints = len(act.Hints)
		act.Hints[len(act.Hints)-1] += " <i>This is the last hint.</i>"
	}
	guesstoken := session.actionToken("guess " + actID)
	hinttoken := session.actionToken("hint " + actID)
	js := MapSI{
		"act":        actID,
		"title":      act.Title,
		"guts":       template.HTML(string(act.Guts)),
		"guesstoken": guesstoken,
		"hinttoken":  hinttoken,
		"hints":      act.Hints[:tas.Hints],
		"solvedP":    tas.SolvedP,
	}
	spewjsonp(w, r, js)
}

// helper func for activity() : show "index.html" page for an activity
func showActPage(w http.ResponseWriter, r *http.Request, session *Session, tid string, context appengine.Context, actID string) {
	key := datastore.NewKey(context, "Activity", actID, 0, nil)
	act := ActivityRecord{}
	err := datastore.Get(context, key, &act)
	if err == datastore.ErrNoSuchEntity {
		w.WriteHeader(http.StatusNotFound)
		showMessage(w, "No such activity", "No such activity", tid, "")
		return
	}
	if err != nil {
		context.Warningf("View act hit ERR %s", err.Error())
		fmt.Fprintf(w, `Got error trying to load activity. %s<br>
                     Things might not work right`, err.Error())
	}
	tas := TAStateRecord{}
	if tid != "" {
		key = datastore.NewKey(context, "TAState", actID+":"+tid, 0, nil)
		err = datastore.Get(context, key, &tas)
		if err != nil && err != datastore.ErrNoSuchEntity {
			context.Warningf("View TAS hit ERR %s", err.Error())
		}
	}
	if tas.Hints > len(act.Hints) {
		tas.Hints = len(act.Hints)
		act.Hints[len(act.Hints)-1] += " <i>This is the last hint.</i>"
	}
	t := template.Must(template.New("").Parse(tActivity))
	guesstoken := session.actionToken("guess " + actID)
	hinttoken := session.actionToken("hint " + actID)
	t.Execute(w, MapSI{
		"Nickname":   actID,
		"PageTitle":  "Activity: " + act.Title,
		"TID":        tid,
		"GuessToken": guesstoken,
		"HintToken":  hinttoken,
		"Title":      act.Title,
		"URL":        act.URL,
		"Guts":       template.HTML(string(act.Guts)),
		"ActInitJSON": MapSI{
			"hints": act.Hints[:tas.Hints],
		},
		"SolvedP":  tas.SolvedP,
		"Solution": strings.ToUpper(act.Solutions[0]),
		"Icon":     actgeticonurl(context, actID),
		"IconLink": actgeticonlink(context, actID),
	})
}

// helper function for activity(). show an actfs thing. E.g., a PDF that
// is part of an activity's "folder"
func showActFS(w http.ResponseWriter, r *http.Request, session *Session, tid string, context appengine.Context, actID string, urlparts []string) {
	afspath := strings.Join(urlparts[2:], "/")
	key := datastore.NewKey(context, "ActFSRecord", afspath, 0, nil)
	afsr := ActFSRecord{}
	err := datastore.Get(context, key, &afsr)
	if err == datastore.ErrNoSuchEntity {
		w.WriteHeader(http.StatusNotFound)
		showMessage(w, "No such file", "No such file", tid, "")
		return
	}
	if err != nil {
		context.Warningf("AFSR GET got ERR %s", err.Error())
	}
	mime := mime.TypeByExtension(path.Ext(afspath))
	if mime != "" {
		w.Header().Set("Content-Type", mime)
	}
	_, err = w.Write(afsr.B)
	if err != nil {
		context.Warningf("AFSR WEB WRITE ERR %s", err.Error())
	}

}

// User guessed at a puzzle.  JSONically respond to that.
func guess(w http.ResponseWriter, r *http.Request) {
	// log the guess. wait a bit. count how many guesses have come in
	// recently (are we dealing w/a bot that guesses many many times?)
	// if many guesses recently, then wait some more.  Finally, respond.

	session, tid := GetAndOrUpdateSession(w, r)
	w.Header().Set("Content-Type", "text/javascript")
	context := appengine.NewContext(r)
	// validate inputs
	actID := r.FormValue("act")
	if actID == "" {
		TLogGuess(context, tid, actID, "malformed", r.FormValue("guess"))
		spewfeedback(w, r, "I don't even recognize this puzzle?")
		return
	}
	guess := normalizeGuess(r.FormValue("guess"))
	if guess == "" {
		guess = "forgottotypesomething"
	}
	complaints := ""
	act := ActivityRecord{}
	key := datastore.NewKey(context, "Activity", actID, 0, nil)
	err := datastore.Get(context, key, &act)
	if err != nil {
		context.Warningf("Guess couldn't load act %s %s", actID, err.Error())
		complaints += "Couldn't load puzzle, got " + err.Error()
	}
	token := session.actionToken("guess " + actID)
	if token != r.FormValue("token") {
		context.Warningf("Got bad XSRF token " + tid + " " + actID)
		complaints += "Something strange is going on; token does not match. "
	}
	if len(complaints) > 0 {
		TLogGuess(context, tid, actID, "malformed", guess)
		spewfeedback(w, r, complaints)
		return
	}

	time.Sleep(time.Millisecond)
	var numRecentGuesses int64 = int64(TLogCountRecentGuesses(context, tid))
	if numRecentGuesses > 10 {
		time.Sleep(time.Millisecond * time.Duration(numRecentGuesses*numRecentGuesses))
	}

	// Enough waiting. figure if they guessed right, wrong, partial, typo.
	for _, s := range act.Solutions {
		if guess == s {
			handleCorrectGuess(w, r, tid, context, actID, act, guess)
			return
		}
	}
	for _, s := range act.Partials {
		split := strings.SplitN(s, " ", 2)
		if guess == split[0] {
			TLogGuess(context, tid, actID, "partial", guess)
			nudge := "I like that guess. You're on the right track!"
			if len(split) > 1 {
				nudge = split[1]
			}
			spewfeedback(w, r, nudge)
			return
		}
	}
	for _, s := range append(act.Solutions, act.Partials...) {
		split := strings.SplitN(s, " ", 2)
		if editDistance(guess, split[0]) <= ((len(guess) + len(split[0])) / 4) {
			TLogGuess(context, tid, actID, "wtfguess", guess)
			spewfeedback(w, r, "Check for typos? That looks a little like something I'd expect.")
			return
		}
	}

	TLogGuess(context, tid, actID, "wtfguess", guess)
	spewfeedback(w, r, strings.ToUpper(guess)+" is not the answer.")
}

// helper function for guess: handle the case where the team guessed
// correctly. if this is the first time the team has solved this puzzle,
// we should unlock some more puzzles.
func handleCorrectGuess(w http.ResponseWriter, r *http.Request, tid string, context appengine.Context, actID string, act ActivityRecord, guess string) {
	nextacts := ""
	for _, nextact := range actgetnext(context, actID) {
		nextacts += " <a class=\"onward\" href=\"/a/" + nextact + "/\">" + nextact + "</a>,"
	}
	if nextacts != "" {
		nextacts = ". Unlocked: " + nextacts[:len(nextacts)-1]
		if tid == "" {
			nextacts = nextacts + "<br><br>If you <a href=\"/loginprompt\">log in</a>, the game can keep track of what you've solved<br>(instead of asking you to re-solve puzzles)"
		}
	}
	feedback := "You solved it! Solution was " + strings.ToUpper(act.Solutions[0]) + nextacts
	var newbadges = map[string]int{}
	tas := TAStateRecord{}
	teamkey := datastore.NewKey(context, "Team", tid, 0, nil)
	takey := datastore.NewKey(context, "TAState", actID+":"+tid, 0, nil)
	if tid != "" {
		datastore.Get(context, takey, &tas)
		if !tas.SolvedP {
			TLogGuess(context, tid, actID, "solve", guess)
			// replaced with a PutMulti
			// tas.TeamID = tid
			// tas.ActID = actID
			// tas.SolvedP = true
			// _, err := datastore.Put(context, key, &tas)
			// if err != nil {
			// 	context.Warningf("Solved but I forgot T %s A %s ERR %s",
			// 		tid, actID, err.Error())
			// }
			datastore.RunInTransaction(context, func(c appengine.Context) (err error) {
				t := TeamRecord{}
				teamkey = datastore.NewKey(context, "Team", tid, 0, nil)
				err = datastore.Get(context, teamkey, &t)
				if err != nil {
					TLog(context, tid, actID, "gypped", fmt.Sprintf("noload %s", err))
					return
				}
				dec := json.NewDecoder(strings.NewReader(t.Tags))
				var points = map[string]int{}
				dec.Decode(&points)
				for _, tag := range act.Tags {
					points[tag] = points[tag] + 1
				}
				jsonbytes, err := json.Marshal(points)
				if err != nil {
					TLog(context, tid, actID, "gypped", "marshalpoints")
					return
				}
				t.Tags = string(jsonbytes)
				dec = json.NewDecoder(strings.NewReader(t.Badges))
				var badges = map[string]int{}
				dec.Decode(&badges)
				ephemeralNewBadges := newBadges(act.Tags, points, badges)
				if len(ephemeralNewBadges) > 0 {
					for k, v := range ephemeralNewBadges {
						badges[k] = v
					}
					jsonbytes, err = json.Marshal(badges)
					if err != nil {
						TLog(context, tid, actID, "gypped", "marshalbadge")
						return
					}
					t.Badges = string(jsonbytes)
				}
				_, err = datastore.Put(context, teamkey, &t)
				if err != nil {
					TLog(context, tid, actID, "gypped", "put")
					return
				}
				newbadges = ephemeralNewBadges
				return
			}, nil)
		}
		if len(newbadges) > 0 {
			feedback = feedback + "<p>&nbsp;<p>You earned "
			if len(newbadges) == 1 {
				feedback = feedback + "a merit badge!"
			} else {
				feedback = feedback + "merit badges!"
			}
			for badge, level := range newbadges {
				TLog(context, tid, actID, "badge", badge)
				feedback = feedback + fmt.Sprintf(`<p>Earned <a target="_blank" href="/b/%s">%s</a> level %d! `, badge, badgeBling[badge].Pretty, level)
			}
		}
	}
	spewjsonp(w, r, MapSI{
		"feedback": feedback,
		"nextacts": actgetnext(context, actID),
	})
	if tid != "" {
		var keys []*datastore.Key
		var tass []TAStateRecord
		if !tas.SolvedP {
			keys = append(keys, takey)
			tas.TeamID = tid
			tas.ActID = actID
			tas.SolvedP = true
			tass = append(tass, tas)
		}
		for _, nextact := range GetLockedActs(context, tid, actgetnext(context, actID)) {
			keys = append(keys,
				datastore.NewKey(context, "TAState", nextact+":"+tid, 0, nil))
			tass = append(tass, TAStateRecord{tid, nextact, false, 0})
		}
		_, err := datastore.PutMulti(context, keys, tass)
		if err != nil {
			context.Warningf("Solved but I forgot something T %s A %s ERR %s",
				tid, actID, err.Error())
		}
	}
}

// User asked for a hint.  So respond to that.
func hint(w http.ResponseWriter, r *http.Request) {
	session, tid := GetAndOrUpdateSession(w, r)
	w.Header().Set("Content-Type", "text/javascript")
	context := appengine.NewContext(r)
	// validate inputs
	actID := r.FormValue("act")
	if actID == "" {
		return
	}

	act := ActivityRecord{}
	akey := datastore.NewKey(context, "Activity", actID, 0, nil)
	err := datastore.Get(context, akey, &act)
	if err != nil {
		context.Warningf("Guess couldn't load act %s %s", actID, err.Error())
		return
	}
	token := session.actionToken("hint " + actID)
	if token != r.FormValue("token") {
		return
	}
	if tid == "" {
		TLogHint(context, tid, actID, 0)
		spewjsonp(w, r, MapSI{
			"hints": []string{
				act.Hints[0] + " <br><i>(If you're not logged in, you can only see the first hint. Log in to enable hints beyond this one.)</i>",
			},
		})
		return
	}
	num, _ := strconv.Atoi(r.FormValue("num"))
	tas := TAStateRecord{}
	taskey := datastore.NewKey(context, "TAState", actID+":"+tid, 0, nil)
	datastore.Get(context, taskey, &tas)
	if num == tas.Hints+1 {
		tas.TeamID = tid
		tas.ActID = actID
		tas.Hints = tas.Hints + 1
		_, err = datastore.Put(context, taskey, &tas)
		if err != nil {
			context.Warningf("Hinted but I forgot T %s A %s ERR %s",
				tid, actID, err.Error())
		}
	}
	if tas.Hints > len(act.Hints) {
		tas.Hints = len(act.Hints)
		act.Hints[len(act.Hints)-1] += " <i>This is the last hint.</i>"
	}
	TLogHint(context, tid, actID, tas.Hints)
	spewjsonp(w, r, MapSI{
		"hints": act.Hints[:tas.Hints],
	})
}

func atokens(w http.ResponseWriter, r *http.Request) {
	session, _ := GetAndOrUpdateSession(w, r)
	actID := r.FormValue("act")
	if actID == "" {
		spewfeedback(w, r, "I don't even recognize this puzzle?")
		return
	}
	spewjsonp(w, r, MapSI{
		"act": actID,
		"g":   session.actionToken("guess " + actID),
		"h":   session.actionToken("hint " + actID),
	})
}

// "Normalize" guesses.  Lower-case, only alpha-numeric.
//   "King George 3" should be "kinggeorge3";
//   "Ocean's 11" should be "oceans11"
func normalizeGuess(guess string) string {
	return scrunch(guess)
}

func populateinteracache(context appengine.Context) {
	key := datastore.NewKey(context, "ActFSRecord", "!/INTERA.txt", 0, nil)
	afsr := ActFSRecord{}
	err := datastore.Get(context, key, &afsr)
	if err != nil {
		return
	}
	lineReader := bufio.NewReader(bytes.NewBuffer(afsr.B))
	prevact := ""
	icon := ""
	iconlink := ""
	arc := ArcCache{}
	for {
		l, isPrefix, err := lineReader.ReadLine()
		line := string(l)
		if isPrefix {
			context.Errorf("ERR CACHING TRUNC LONG LINE %s", line)
		}
		if err == io.EOF {
			if arc.Type == "sequence" {
				arccache[arc.Nick] = arc
			}
			break
		}
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "NICK") {
			if arc.Type == "sequence" {
				arccache[arc.Nick] = arc
			}
			arc = ArcCache{}
			arc.Nick = strings.TrimSpace(line[4:])
			prevact = ""
			icon = ""
			iconlink = ""
		}
		if strings.HasPrefix(line, "TYPE") {
			arc.Type = strings.TrimSpace(line[4:])
		}
		if strings.HasPrefix(line, "TITLE") {
			arc.Title = strings.TrimSpace(line[5:])
		}
		if strings.HasPrefix(line, "ICONLINK") {
			iconlink = strings.TrimSpace(line[8:])
		}
		if strings.HasPrefix(line, "ICON ") {
			icon = strings.TrimSpace(line[4:])
			if arc.Icon == "" {
				arc.Icon = icon
			}
		}
		if strings.HasPrefix(line, "A ") {
			ic := interacache["!AUTO"]
			act := scrunch(line[2:])
			arc.Act = append(arc.Act, act)
			ic.Next = append(ic.Next, act)
			interacache["!AUTO"] = ic
			ic = interacache[act]
			if icon != "" && ic.Icon == "" {
				ic.Icon = icon
				interacache[act] = ic
			}
			if iconlink != "" && ic.Iconlink == "" {
				ic.Iconlink = iconlink
				interacache[act] = ic
			}
			prevact = act
		}
		if strings.HasPrefix(line, "L ") {
			if prevact == "" {
				continue
			}
			ic := interacache[prevact]
			act := scrunch(line[2:])
			arc.Act = append(arc.Act, act)
			ic.Next = append(ic.Next, act)
			interacache[prevact] = ic
			ic = interacache[act]
			if icon != "" && ic.Icon == "" {
				ic = interacache[act]
				ic.Icon = arc.Icon
				interacache[act] = ic
			}
			if iconlink != "" && ic.Iconlink == "" {
				ic = interacache[act]
				ic.Iconlink = iconlink
				interacache[act] = ic
			}
		}
		if strings.HasPrefix(line, "X ") {
			prevact = scrunch(line[2:])
			arc.Act = append(arc.Act, prevact)
			if icon != "" {
				ic := interacache[prevact]
				if ic.Icon == "" {
					ic.Icon = icon
				}
				interacache[prevact] = ic
			}
			if iconlink != "" {
				ic := interacache[prevact]
				if ic.Iconlink == "" {
					ic.Iconlink = iconlink
				}
				interacache[prevact] = ic
			}
		}
		if strings.HasPrefix(line, "N ") {
			act := scrunch(line[2:])
			arc.Act = append(arc.Act, act)
			if prevact != "" {
				ic := interacache[prevact]
				ic.Next = append(ic.Next, act)
				interacache[prevact] = ic
			}
			if icon != "" {
				ic := interacache[act]
				if ic.Icon == "" {
					ic.Icon = icon
				}
				interacache[act] = ic
			}
			if iconlink != "" {
				ic := interacache[act]
				if ic.Iconlink == "" {
					ic.Iconlink = iconlink
				}
				interacache[act] = ic
			}
			prevact = act
		}
	}
}

func actgetnext(context appengine.Context, from string) (to []string) {
	to = interacache[from].Next
	if len(to) > 0 {
		return
	}
	if len(interacache["!AUTO"].Next) == 0 {
		populateinteracache(context)
	}
	to = interacache[from].Next
	return
}

func actgeticonurl(context appengine.Context, act string) (iconurl string) {
	iconurl = interacache[act].Icon
	if iconurl != "" {
		return
	}
	if len(interacache["!AUTO"].Next) == 0 {
		populateinteracache(context)
	}
	iconurl = interacache[act].Icon
	return
}

func actgeticonlink(context appengine.Context, act string) (iconlink string) {
	iconlink = interacache[act].Iconlink
	if iconlink != "" {
		return
	}
	if len(interacache["!AUTO"].Next) == 0 {
		populateinteracache(context)
	}
	iconlink = interacache[act].Iconlink
	return
}

func fetcharc(context appengine.Context, nick string) (arc ArcCache) {
	arc = arccache[nick]
	if arc.Nick == nick {
		return
	}
	if len(interacache["!AUTO"].Next) == 0 {
		populateinteracache(context)
	}
	arc = arccache[nick]
	return
}
