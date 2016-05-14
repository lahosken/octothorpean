package octo

import (
	"appengine"
	"appengine/datastore"
	"appengine/mail"
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"text/template"
	"time"
)

func mailedTeamRecentlyP(context appengine.Context, team TeamRecord) bool {
	monthAgo := time.Now().AddDate(0, -1, -1)
	q := datastore.NewQuery("TLog").Filter("TeamID=", team.ID).Order("-Created")
	for iter := q.Run(context); ; {
		var tlr TLogRecord
		_, err := iter.Next(&tlr)
		if err == datastore.Done {
			return false
		}
		if err != nil {
			// we got a weird error. play it safe; don't send mail
			return true
		}
		if tlr.Created.Before(monthAgo) {
			return false
		}
		if tlr.Verb == "mailed" {
			return true
		}
	}
	return false
}

func alreadySolvedP(context appengine.Context, team TeamRecord, actID string) bool {
	tas := TAStateRecord{}
	// if I cared about being efficient, this would be a GetMulti... oh well
	key := datastore.NewKey(context, "TAState", actID+":"+team.ID, 0, nil)
	datastore.Get(context, key, &tas)
	return tas.SolvedP
}

func madlib(team TeamRecord, snippets []string) string {
	const letter = `
Hello excellent Team {{.Team.ID}} !

This friendly announcement points out some
Octothorpean puzzles you haven't solved yet.
They might be tough or perhaps well-hidden.
{{range .Snippets}}
{{.}}
{{end}}

Enjoy!

# # #

(If you don't want future announcements, edit your team info:
http://www.octothorpean.org/editteamprompt
un-check "Announcements?" and press the Update button.)
`
	t := template.Must(template.New("letter").Parse(letter))
	buf := new(bytes.Buffer)
	t.Execute(buf, MapSI{
		"Team":     team,
		"Snippets": snippets,
	})
	return buf.String()
}

// Mail a "straggler" to encourage them to come back.
//
// To make a sensible mail, we want a description for unlocked-but-not-solved
// puzzles. We have "blurbs" for ~17 puzzles. So... pick one of those. Pick
// a team that has unlocked but not solved the puzzle, has other "blurbed"
// puzzles unlocked but that we haven't already mailed recently. Mail that team.
func cronmailstraggler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<html><li>hello, world`)
	context := appengine.NewContext(r)
	baq := datastore.NewQuery("Activity").Filter("Blurb >", "")
	var blurbedActs []ActivityRecord
	_, err := baq.GetAll(context, &blurbedActs)
	if err != nil {
		fmt.Fprintf(w, `Error fetching Blurbed acts: %v`, err)
		return
	}
	fmt.Fprintf(w, `<li>First act %v`, blurbedActs[0].Nickname)
	oneBlurbedAct := blurbedActs[rand.Intn(len(blurbedActs))].Nickname
	fmt.Fprintf(w, `<li>One act %v`, oneBlurbedAct)
	taq := datastore.NewQuery("TAState").Filter("ActID =", oneBlurbedAct)
	var oneActStates []TAStateRecord
	_, err = taq.GetAll(context, &oneActStates)
	if err != nil {
		fmt.Fprintf(w, `<li>Error fetching TAstates for act act %v: %v`, oneBlurbedAct, err)
		return
	}
	if len(oneActStates) == 0 {
		fmt.Fprintf(w, `<li>Fetched zero TAstates for act act %v`, oneBlurbedAct)
		return
	}
	for tries := 0; tries < 100; tries++ {
		oneActState := oneActStates[rand.Intn(len(oneActStates))]
		if oneActState.SolvedP {
			continue
		}
		fmt.Fprintf(w, `<li>Load team %v`, oneActState.TeamID)
		teamKey := datastore.NewKey(context, "Team", oneActState.TeamID, 0, nil)
		team := TeamRecord{}
		err = datastore.Get(context, teamKey, &team)
		if err != nil {
			fmt.Fprintf(w, `<li>Error loading team: %v`, err)
			continue
		}
		if team.AnnounceOK == 0 {
			fmt.Fprintf(w, `<li>Team doesn't want mail, continuing`)
			continue
		}
		// team is looking pretty solid. What all unlocked-not-solved
		// puzzles does it have, anyhow? Any more with blurbs?
		var unsolvedBlurbs []string
		tmaq := datastore.NewQuery("TAState").Filter("TeamID =", oneActState.TeamID)
		tmaqi := tmaq.Run(context)
		for {
			var tas TAStateRecord
			_, err = tmaqi.Next(&tas)
			if err == datastore.Done {
				break
			}
			if err != nil {
				fmt.Fprintf(w, `<li>Error loading team states %v`, err)
				break
			}
			if tas.SolvedP {
				continue
			}
			fmt.Fprintf(w, `<li>saw unsolved puzzle %v`, tas.ActID)
			for _, blurbedAct := range blurbedActs {
				if blurbedAct.Nickname != tas.ActID {
					continue
				}
				unsolvedBlurbs = append(unsolvedBlurbs,
					"+ "+blurbedAct.Title+": "+blurbedAct.Blurb+
						"\n    http://www.octothorpean.org/a/"+blurbedAct.Nickname+"/")
			}
		}
		if len(unsolvedBlurbs) < 3 {
			fmt.Fprintf(w, `<li>Not so many unsolved for this team, keep looking`)
			continue
		}
		if mailedTeamRecentlyP(context, team) {
			fmt.Fprintf(w, `<li>Already mailed that team recently, keep looking`)
			continue
		}
		ml := madlib(team, unsolvedBlurbs)
		msg := &mail.Message{
			Sender: "Octothorpean <octothorpean@gmail.com>",
			To:      team.EmailList,
			Subject: "# More ## Puzzles ###",
			Body:    ml,
			Bcc:     []string{"lahosken@gmail.com"},
		}

		if err := mail.Send(context, msg); err != nil {
			context.Errorf("Couldn't send email to %s: %v", team.ID, err)
			continue
		}

		TLog(context, team.ID, "", "mailed", "")

		fmt.Fprintf(w, `<li>Breaking 2 Electric Boogaloo <pre>%v</pre>`, unsolvedBlurbs[0])
		break
	}
}
