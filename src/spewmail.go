package octo

import (
	"appengine"
	"appengine/datastore"
	"appengine/mail"
	"bytes"
	"math/rand"
	"net/http"
	"text/template"
	"time"
)

// not much here yet...

func getMailableTeams(context appengine.Context) (l []TeamRecord) {
	q := datastore.NewQuery("Team")
	for iter := q.Run(context); ; {
	    var team TeamRecord
	    _, err := iter.Next(&team)
	    if err == datastore.Done {
			break
		}
     	if len(team.EmailList) < 1  {
			continue
		}
		if rand.Intn(10) > 0 {
			continue
		}
		if team.AnnounceOK != 1 {
			continue
		}
		if team.Tags == "" {
			continue
		}
		l = append(l, team)
	}
	return
}

func mailedTeamRecentlyP(context appengine.Context, team TeamRecord) (bool) {
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

func alreadySolvedP(context appengine.Context, team TeamRecord, actID string) (bool) {
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
They might be new, they might be tough, or
perhaps just well-hidden.

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
		"Team": team,
		"Snippets": snippets,
	})
	return buf.String()
}

func snippets201312(context appengine.Context, team TeamRecord) (snippets []string) {
	type PS struct {
		p string
		s string
	}
	table := []PS {
		{
			"onlynumbers",
			"NEW PUZZLE http://www.octothorpean.org/a/onlynumbers/ has numbers",
		},
		{
			"porr",
			"NEW PUZZLE http://www.octothorpean.org/a/porr/ has a poem",
		},
		{
			"gasc",
			"NEW PUZZLE http://www.octothorpean.org/a/gasc/ has no poem",
		},
		{
			"connect",
			"Puzzle http://www.octothorpean.org/a/connect/ is islandic",
		},
		{
			"toomuch",
			"Puzzle http://www.octothorpean.org/a/toomuch/ , courtesy of Puzzled Pint",
		},
		{
			"fourwinds",
			"Old Demo Puzzle http://www.octothorpean.org/a/fourwinds/",
		},
	}
	for _, v := range(table) {
		if (!alreadySolvedP(context, team, v.p)) {
			snippets = append(snippets, v.s)
		}
		if len(snippets) > 3 { break }
	}
	return
}

func adminspewmail(w http.ResponseWriter, r *http.Request) {
	aid := checkAdminLogin(w, r)
	if aid == "" {
		spewjsonp(w, r, "You really need to log in. Sorry about that.")
		return
	}
	context := appengine.NewContext(r)
	teams := getMailableTeams(context)
	count := 0
    for _, team := range(teams) {
		context.Infof("  team %s", team.ID)
		if mailedTeamRecentlyP(context, team) {
			context.Infof("  ! RECENT")
			continue
		}
		snippets := snippets201312(context, team)
		if len(snippets) < 1 {
			context.Infof("  ! SNIPPETS")
			continue
		}

		ml := madlib(team, snippets)
		msg := &mail.Message{
            Sender:  "Octothorpean <octothorpean@gmail.com>",
            To:      team.EmailList,
            Subject: "# more ## puzzles ###",
            Body:    ml,
			Bcc: []string{"lahosken@gmail.com"},
        }
		if err := mail.Send(context, msg); err != nil {
            context.Errorf("Couldn't send email to %s: %v", team.ID, err)
			continue
        }
		
		TLog(context, team.ID, "", "mailed", "")
		count = count + 1
		if count > 9 { break }
	}
	spewjsonp(w, r, count)
}
