package octo

/**
 * Team-visible "dashboard" of what other teams are doing.
 */

import (
	"appengine"
	"appengine/datastore"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"
)

type tidbit struct {
	T int64
	M string
}

type GossipCache struct {
	time   time.Time
	record []TLogRecord
}

var gossipCache GossipCache

func fetchGossip(context appengine.Context) []TLogRecord {
	// if cache is not fresh...
	if gossipCache.time.Add(time.Duration(time.Minute)).Before(time.Now()) {
		q := datastore.NewQuery("TLog").Order("-Created").Limit(100)
		var fetched []TLogRecord = make([]TLogRecord, 100)
		// TODO q.GetAll didn't work in r58
		for iter := q.Run(context); ; {
			var tlr TLogRecord
			_, err := iter.Next(&tlr)
			if err == datastore.Done {
				break
			}
			if err != nil {
				context.Warningf("Gossip iter ERR %s", err.Error())
				break
			}
			if tlr.TeamID == "" { // TODO howto filter for this in query?
				continue
			}
			fetched = append(fetched, tlr)
		}
		gossipCache = GossipCache{time.Now(), fetched}
	}
	return gossipCache.record
}

func gossip(w http.ResponseWriter, r *http.Request) {
	_, tid := GetAndOrUpdateSession(w, r)
	w.Header().Set("Content-Type", "text/javascript")
	context := appengine.NewContext(r)

	fetched := fetchGossip(context)

	alreadySet := make(map[string]bool) // say something once, why say it again?
	l := []tidbit{}

	for _, tlr := range fetched {
		if tlr.TeamID == "" {
			continue
		}
		t := "Your team"
		a := fmt.Sprintf(`<a href="/a/%s/">%s</a>`,
			html.EscapeString(tlr.ActID), html.EscapeString(tlr.ActID))
		if tlr.TeamID != tid {
			t = fmt.Sprintf(`Team <a href="/team/%s">%s</a>`,
				html.EscapeString(url.QueryEscape(url.QueryEscape(tlr.TeamID))),
				html.EscapeString(tlr.TeamID))
			if tlr.ActID != r.FormValue("act") {
				a = "something" + strings.Repeat(" ", len(tlr.ActID)%4)
			}
		}
		n := ""
		switch {
		case tlr.Verb == "badge":
			n = fmt.Sprintf(` earned badge : <a target="_blank" href="/b/%s">%s</a>`,
				tlr.Notes, badgeBling[tlr.Notes].Pretty)
		case tlr.Verb == "solve":
			n = " solved " + a
			alreadySet[t+" worked on "+a] = true
			alreadySet[t+" was active"] = true
			alreadySet[t+" guessed at "+a+" with "+tlr.Guess] = true
		case tlr.Verb == "guess":
			n = " worked on " + a
			if tlr.TeamID == tid {
				n = " guessed at " + a + " with " + tlr.Guess
			}
			alreadySet[t+" was active"] = true
		case tlr.Verb == "hint":
			n = " worked on " + a
			if tlr.TeamID == tid {
				n = " got a hint for " + a
			}
			alreadySet[t+" was active"] = true
		case tlr.Verb == "login":
			n = " was active"
			if tlr.TeamID == tid {
				n = " logged in"
			}
		}
		if n == "" || alreadySet[t+n] {
			continue
		}
		alreadySet[t+n] = true
		l = append(l, tidbit{
			T: tlr.Created.Unix() * 1000,
			M: t + n,
		})
		if len(l) >= 20 {
			break
		}
	}
	spewjsonp(w, r, MapSI{"gossip": l})
}

// Get gossip for one team. Handy for displaying on their profile page.
func getTeamGossip(context appengine.Context, tid string) (out []tidbit) {
	alreadySet := make(map[string]bool) // say something once, why say it again?
	q := datastore.NewQuery("TLog").Order("-Created").Filter("TeamID=", tid)
	for iter := q.Run(context); ; { // TODO GetAll didn't work in r58
		var tlr TLogRecord
		_, err := iter.Next(&tlr)
		if err == datastore.Done {
			break
		}
		if err != nil {
			context.Warningf("TeamGossip iter ERR %s", err.Error())
			break
		}
		// TODO
		a := "something" + strings.Repeat(" ", len(tlr.ActID)%4)
		// ideally, we'd be specific if viewing-team
		// knew of puzzle's existence and say "something" if viewing-team
		// didn't know of puzzle's existence. TODO
		// a := fmt.Sprintf(`<a href="/a/%s/">%s</a>`,
		//	html.EscapeString(tlr.ActID), html.EscapeString(tlr.ActID))

		n := ""
		switch {
		case tlr.Verb == "badge":
			n = fmt.Sprintf(` earned badge : <a target="_blank" href="/b/%s">%s</a>`,
				tlr.Notes, badgeBling[tlr.Notes].Pretty)
		case tlr.Verb == "solve":
			n = "Solved " + a
			alreadySet["Worked on "+a] = true
			alreadySet["Was active"] = true
		case tlr.Verb == "guess":
			n = "Worked on " + a
			alreadySet["Was active"] = true
		case tlr.Verb == "hint":
			n = "Worked on " + a
			alreadySet["Was active"] = true
		case tlr.Verb == "login":
			n = "Was active"
		}
		if n == "" || alreadySet[n] {
			continue
		}
		alreadySet[n] = true
		out = append(out, tidbit{
			T: tlr.Created.Unix() * 1000,
			M: n,
		})
		if len(out) > 50 {
			break
		}
	}
	return
}

func dashboard(w http.ResponseWriter, r *http.Request) {
	_, tid := GetAndOrUpdateSession(w, r)
	template.Must(template.New("").Parse(tDashboard)).Execute(w, struct {
		PageTitle string
		TID       string
	}{
		PageTitle: "Dashboard",
		TID:       tid,
	})
}
