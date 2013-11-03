package octo

import (
	"appengine"
	"appengine/datastore"
	"net/http"
//	"appengine/mail"
)

// not much here yet...

func getMailableTeams(context appengine.Context) (l []string) {
	var teams []TeamRecord
	q:= datastore.NewQuery("Team").Order("-LastSeen")
	_, _ = q.GetAll(context, &teams)
	for _, team := range teams {
		// REMIND : for starters, test by only mailing teams which are ME
		if len(team.EmailList) != 1 { continue }
		if team.EmailList[0] != "lahosken@gmail.com" { continue }
		// REMIND
		if team.AnnounceOK != 1 { continue }
		l = append(l, team.ID)
	}
	return
}

func adminspewmail(w http.ResponseWriter, r *http.Request) {
}