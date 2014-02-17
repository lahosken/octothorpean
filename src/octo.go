package octo

import (
	"appengine"
	"appengine/datastore"
	"encoding/json"
	"fmt"
	//  "log"
	"net/http"
	"strings"
	"text/template"
	"unicode"

//	"github.com/mjibson/appstats"
)

func init() {
	// top screen
	http.HandleFunc("/", topscreen)
	// activities and groups of activities
	// http.HandleFunc("/a/", activity)
	http.HandleFunc("/a/", activity)
	http.HandleFunc("/a.json", activityjson)
	http.HandleFunc("/arc/", arc)
	http.HandleFunc("/arc.json", arcjson)
	http.HandleFunc("/atokens", atokens)
	http.HandleFunc("/b/", badgeprofile)
	http.HandleFunc("/editteamprompt", editteamprompt)
	http.HandleFunc("/editteam", editteam)
	http.HandleFunc("/gossip", gossip)
	http.HandleFunc("/guess", guess)
	http.HandleFunc("/hint", hint)
	http.HandleFunc("/loginprompt", loginprompt)
	http.HandleFunc("/login", login)
	http.HandleFunc("/login.json", loginjson)
	http.HandleFunc("/logout", logout)
	// team and team-login stuff
	http.HandleFunc("/registerprompt", registerprompt)
	http.HandleFunc("/register", register)
	http.HandleFunc("/resetpasswordprompt", resetpasswordprompt)
	http.HandleFunc("/resetpassword", resetpassword)
	http.HandleFunc("/team/", teamprofile)
	http.HandleFunc("/whosplaying", dashboard)
	// admin
	http.HandleFunc("/admin/", adminmenu)
	http.HandleFunc("/admin/gossip", admingossip)
	http.HandleFunc("/admin/gossip.json", admingossipjson)
	http.HandleFunc("/admin/logs", adminlogs)
	http.HandleFunc("/admin/wtf", adminwtflogs)
	http.HandleFunc("/admin/login", adminlogin)
	http.HandleFunc("/admin/upload", adminupload)
	http.HandleFunc("/admin/digestupload", digestupload)
	http.HandleFunc("/admin/uploadintera", adminuploadintera)
	http.HandleFunc("/admin/uploadprompt", adminuploadprompt)
	http.HandleFunc("/admin/editadmin", admineditadmin)
	http.HandleFunc("/admin/editactivity", admineditactivity)
	http.HandleFunc("/admin/teamspreadsheet", adminteamspreadsheet)
	http.HandleFunc("/admin/spewmail", adminspewmail)
	http.HandleFunc("/admin/maillist", adminmaillist)
	http.HandleFunc("/admin/dumpteamlogs.tsv", admindumpteamlogs)
	http.HandleFunc("/admin/editteam", admineditteam)
	http.HandleFunc("/admin/cleanteam", admincleanteam)
	// morlocks
	http.HandleFunc("/cron/weehours", cronweehours)
	// wombat
	if WOMBAT_ENABLE == true {
		http.HandleFunc("/wombat/act.json", wombatact)
		http.HandleFunc("/wombat/arc.json", wombatarc)
	}
}

func topscreen(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `<center><b>404 not found</b><br>
                        <a href="/">Try something else</a></center>`)
		return
	}
	_, tid := GetAndOrUpdateSession(w, r)
	links := []string{}
	if tid != "" {
		context := appengine.NewContext(r)
		q := datastore.NewQuery("TLog").Filter("TeamID=", tid).Order("-Created")
		for iter := q.Run(context); ; {
			var tlr TLogRecord
			_, err := iter.Next(&tlr)
			if err == datastore.Done {
				break
			}
			if err != nil {
				continue
			}
			if tlr.Verb != "solve" {
				continue
			}
			followers := actgetnext(context, tlr.ActID)
			if len(followers) < 3 {
				continue
			}
			links = append(links, tlr.ActID)
			if len(links) > 3 {
				break
			}
		}
	}
	template.Must(template.New("").Parse(tTop)).Execute(w, MapSI{
		"PageTitle": "Octothorpean Order",
		"TID":       tid,
		"Links":     links,
	})
}

// Levenshtein-ish edit distance. "cat"->"car" is small, "cat"->"doggie" is more
// Hoisted from James Keane https://gist.github.com/1069374 , altered to taste
func editDistance(a, b string) int {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	d := make([][]int, len(a)+1)
	for i, _ := range d {
		d[i] = make([]int, len(b)+1)
		d[i][0] = i
	}

	for i, _ := range d[0] {
		d[0][i] = i
	}

	for i := 1; i < len(d); i++ {
		for j := 1; j < len(d[0]); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}

			min := d[i-1][j-1] + cost
			min2 := d[i][j-1] + 1
			if min2 < min {
				min = min2
			}
			min3 := d[i-1][j] + 1
			if min3 < min {
				min = min3
			}
			d[i][j] = min
		}
	}

	return d[len(a)][len(b)]
}

func cronweehours(w http.ResponseWriter, r *http.Request) {
	context := appengine.NewContext(r)
	CleanupSessions(context)
	CleanupTeamLogs(context)
}

// Lower-case string and toss out everything that isn't alphanumeric
func scrunch(s string) string {
	lcalnum := func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return -1
	}
	return strings.Map(lcalnum, s)
}

// Given an interface, JSON it perhaps wrapped in a callback function
func spewjsonp(w http.ResponseWriter, r *http.Request, v interface{}) {
	j, _ := json.MarshalIndent(v, " ", "  ")
	js := string(j)
	if r.FormValue("callback") != "" {
		js = r.FormValue("callback") + "(" + js + ")"
	}
	fmt.Fprint(w, string(js))
}

// JSONic and templated like map[string]interface{}, so we use it plenty.
// Let's abbreviate it:
type MapSI map[string]interface{}

func spewfeedback(w http.ResponseWriter, r *http.Request, s string) {
	spewjsonp(w, r, map[string]string{"feedback": s})
}
