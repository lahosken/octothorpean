package octo

/**
 * Storing team data.  Handling login, logout, team creation. Lots of UI.
 */

import (
	"appengine"
	"appengine/datastore"
	"appengine/mail"
	"appengine/memcache"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"
)

// Present a login form
func loginprompt(w http.ResponseWriter, r *http.Request) {
	_, tid := GetAndOrUpdateSession(w, r)
	if tid != "" {
		showMessage(w, "Error: already logged in",
			"You are already logged in", tid, "")
		return
	}
	template.Must(template.New("").Parse(tLoginPrompt)).Execute(w, map[string]string{
		"PageTitle": "Please log in",
		"Team":      r.FormValue("team"),
		// TODO URL escaping would be pretty sweet
		"TeamURL": url.QueryEscape(r.FormValue("team")),
	})
}

// Handle login attempt
func login(w http.ResponseWriter, r *http.Request) {
	session, tid := GetAndOrUpdateSession(w, r)
	if tid != "" {
		showMessage(w, "Error: already logged in",
			"You are already logged in.", tid, "")
		return
	}
	context := appengine.NewContext(r)
	tid = strings.TrimSpace((r.FormValue("team")))
	t := getTeam(context, tid)
	if t == nil {
		showNoSuchTeam(w, r, context, tid)
		return
	}
	enteredPassword := strings.TrimSpace(r.FormValue("password"))
	if enteredPassword != t.Password {
		template.Must(template.New("").Parse(tLoginFailedPassword)).Execute(w,
			map[string]string{
				"PageTitle": "Error: Password did not match!",
				"Team":      tid,
				// URL escaping would be pretty sweet
				"TeamURL": url.QueryEscape(r.FormValue("team")),
			})

		return
	}
	session.loginSession(context, tid)
	redirToTopPage(w)
	TLog(context, tid, "", "login", "")
}

func loginjson(w http.ResponseWriter, r *http.Request) {
	session, tid := GetAndOrUpdateSession(w, r)
	if tid != "" {
		js := MapSI{
			"success":  false,
			"team":     tid,
			"message":  "Already logged in!",
			"similars": []string{},
		}
		spewjsonp(w, r, js)
		return
	}
	context := appengine.NewContext(r)
	tid = strings.TrimSpace((r.FormValue("team")))
	t := getTeam(context, tid)
	if t == nil {
		js := MapSI{
			"success":  false,
			"team":     "",
			"message":  "No such team!",
			"similars": getTeamIDsSimilarTo(context, tid, 5),
		}
		spewjsonp(w, r, js)
		return
	}
	enteredPassword := strings.TrimSpace(r.FormValue("password"))
	if enteredPassword != t.Password {
		js := MapSI{
			"success":  false,
			"team":     tid,
			"message":  "Password did not match!",
			"similars": []string{},
		}
		spewjsonp(w, r, js)
		return
	}
	session.loginSession(context, tid)
	TLog(context, tid, "", "login", "")
	js := MapSI{
		"success":  true,
		"team":     tid,
		"message":  "OK.",
		"similars": []string{},
	}
	spewjsonp(w, r, js)
}

// user tried to log in, put to a nonexistent team. Show login screen
// with some suggestions.
func showNoSuchTeam(w http.ResponseWriter, r *http.Request, context appengine.Context, tid string) {
	similars := getTeamIDsSimilarTo(context, tid, 5)
	template.Must(template.New("").Parse(tLoginFailedNoSuchTeam)).Execute(w, MapSI{
		"PageTitle": "No such team.",
		"Team":      tid,
		// TODO is there an URL-escape in template? (not yet, but someday?)
		"Description": url.QueryEscape(tid),
		"Similars":    similars,
		"TID":         "",
		"TeamURL":     url.QueryEscape(r.FormValue("team")),
	})
}

func redirToTopPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
      <meta http-equiv="refresh" content="0;url=/">
      <a href="/">On to the welcome screen</a>
    `)
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := GetAndOrUpdateSession(w, r)
	context := appengine.NewContext(r)
	session.loginSession(context, "")
	redirToTopPage(w)
	TLog(context, "", "", "logout", "")
}

func registerprompt(w http.ResponseWriter, r *http.Request) {
	_, tid := GetAndOrUpdateSession(w, r)
	if tid != "" {
		showMessage(w, "Error: already logged in",
			"You are already logged in", tid, "")
		return
	}
	r.FormValue("Fake") // TODO kludge to force r.Form["m"] to be created
	emailList := r.Form["m"]
	emailList = append(emailList, "", "", "", "")[:4]
	template.Must(template.New("").Parse(tRegisterPrompt)).Execute(w, MapSI{
		"PageTitle":   "Register your team",
		"TID":         "",
		"Team":        r.FormValue("team"),
		"EmailList":   emailList,
		"Description": r.FormValue("description"),
		"TeamURL":     url.QueryEscape(r.FormValue("team")),
	})
}

func register(w http.ResponseWriter, r *http.Request) {
	_, tid := GetAndOrUpdateSession(w, r)
	if tid != "" {
		showMessage(w, "Error: already logged in",
			"You are already logged in", tid, "")
		return
	}
	r.FormValue("Fake") // TODO kludge to force r.Form created (?not needed?)
	undoURL := "/registerprompt?" + r.Form.Encode()

	tid = normalizeNewTeamName(r.FormValue("team"))
	tid = strings.TrimSpace(tid)
	description := r.FormValue("description")
	if len(description) > 150 {
		description = description[:150]
	}
	emailList := []string{}
	for _, e := range r.Form["m"] {
		if len(strings.TrimSpace(e)) > 0 {
			emailList = append(emailList, e)
		}
	}

	complaintsHTML := ""
	if len(tid) < 4 {
		complaintsHTML += "<p>Please choose a longer Team name"
	}
	if len(emailList) == 0 {
		complaintsHTML += "<p>Please specify at least one email address. (We need it to send the password to)"
	}
	context := appengine.NewContext(r)
	similars := getTeamIDsSimilarTo(context, tid, 0)
	if len(similars) > 0 {
		complaintsHTML += `<p>Unfortunately, that team name would be too
                              similar to some other team name(s). Please
                              choose a different-er name. Similar name(s):<ul>`
		for _, s := range similars {
			complaintsHTML += "<li>" + html.EscapeString(s)
		}
		complaintsHTML += "</ul>"
	}
	key := datastore.NewKey(context, "Team", tid, 0, nil)
	t := TeamRecord{}
	err := datastore.Get(context, key, &t)
	if err == nil {
		complaintsHTML += "<p>Please choose another team name. That team already exists."
	}
	if err != nil && err != datastore.ErrNoSuchEntity {
		context.Warningf("Strange t confirm TID=%s ERR=%s", tid, err.Error())
	}
	if len(complaintsHTML) > 0 {
		template.Must(template.New("").Parse(tRegisterHitch)).Execute(w, map[string]string{
			"PageTitle": "Hitch registering team",
			"Message":   complaintsHTML,
			"GoBack":    undoURL,
		})
		return
	}
	announceok := 0
	if r.FormValue("announceok") != "" {
		announceok = 1
	}

	newTeamRecord := TeamRecord{
		ID:          tid,
		Created:     time.Now(),
		LastSeen:    time.Now(),
		EmailList:   emailList,
		Password:    generateRandomPassword(),
		AnnounceOK:  announceok,
		Description: description}
	_, err = datastore.Put(context, key, &newTeamRecord)
	if err != nil {
		context.Errorf("Couldn't save team TID=%s E0=%s ERR=%s", tid, emailList[0], err.Error())
		showMessage(w, "Error: something went wrong",
			"Something unusual happened when we tried to 'save' your team data: "+err.Error(), "", undoURL)
		return
	}
	// Unlock the first set of puzzles
	for _, nextact := range actgetnext(context, "!AUTO") {
		UnlockAct(context, tid, nextact)
	}
	memcache.Delete(context, "Teams/IDList")
	msg := &mail.Message{
		Sender:  "octothorpean@gmail.com",
		To:      emailList,
		Subject: "Confirm your registration",
		Body: fmt.Sprintf(`
Thank you for creating an account on octothorpean.org!

Your team name is: %s
Your password is: %s 

Login at https://octothorpean.org/loginprompt?team=%s    

# # # Have fun! # # #

(You can send comments, typo reports, etc.
to this octothorpean@gmail.com address.)
    `, newTeamRecord.ID, newTeamRecord.Password, url.QueryEscape(newTeamRecord.ID))}
	if err := mail.Send(context, msg); err != nil {
		context.Errorf("Accont create couldn't send mail TID=%s E0=%s ERR=%s", tid, emailList[0], err.Error())
		showMessage(w, "Error: mail fail",
			"Something unusual happened when we tried to mail your password: "+err.Error()+" If your confirmation mail doesn't show up, you might try resetting your password.", "", "/loginprompt")
		return
	}
	showMessage(w, "Check your mail!",
		"Account created! Check your mail for the password.", "",
		fmt.Sprintf("/loginprompt?team=%s", url.QueryEscape(newTeamRecord.ID)))
}

func resetpasswordprompt(w http.ResponseWriter, r *http.Request) {
	session, tid := GetAndOrUpdateSession(w, r)
	if tid != "" && tid != r.FormValue("team") {
		showMessage(w, "Error: already logged in", "You are already logged in",
			tid, "")
		return
	}
	token := session.actionToken("reset password")
	template.Must(template.New("").Parse(tResetPasswordPrompt)).Execute(w, map[string]string{
		"PageTitle": "Reset Password?",
		"Team":      r.FormValue("team"),
		// URL escaping would be pretty sweet
		"TeamURL": url.QueryEscape(r.FormValue("team")),
		"Token":   token,
		"TID":     tid,
	})
}

func resetpassword(w http.ResponseWriter, r *http.Request) {
	session, tid := GetAndOrUpdateSession(w, r)
	if tid != "" && tid != r.FormValue("team") {
		showMessage(w, "Error: already logged in", "You are already logged in",
			tid, "")
		return
	}
	tid = strings.TrimSpace(r.FormValue("team"))
	token := session.actionToken("reset password")
	context := appengine.NewContext(r)
	if token != r.FormValue("token") {
		context.Errorf("passwd reset TOKEN TID=%s IP=%s", tid, r.RemoteAddr)
		showMessage(w, "Error: token didn't match",
			`Something strange is going on. It looks like you want to reset
             a password. But the token didn't match.`, tid, "")
		return
	}
	key := datastore.NewKey(context, "Team", tid, 0, nil)
	t := TeamRecord{}
	err := datastore.Get(context, key, &t)
	if err != nil {
		context.Errorf("Passwd reset couldn't read team TID=%s ERR=%s", tid, err.Error())
		showMessage(w, "Error: Couldn't read team data",
			`Something strange happened. Couldn't read team data: `+
				err.Error(), tid, "")
		return
	}
	t.Password = generateRandomPassword()
	t.LastSeen = time.Now()
	_, err = datastore.Put(context, key, &t)
	if err != nil {
		context.Errorf("Passwd reset couldn't save team TID=%s ERR=%s", tid, err.Error())
		showMessage(w, "Error: Couldn't save new password",
			`Something strange happened. Couldn't save new password: `+
				err.Error(), tid, "")
		return
	}
	msg := &mail.Message{ // TODO this will change a lot
		Sender:  "octothorpean@gmail.com",
		To:      t.EmailList,
		Subject: "Password reset",
		Body: fmt.Sprintf(`
    Thank you for resetting a password on octothorpean.org!

    Your team name is: %s
    Your password is: %s 

    Have fun!
    `, t.ID, t.Password)}
	if err := mail.Send(context, msg); err != nil {
		context.Errorf("Passwd reset couldn't send mail TID=%s E0=%s ERR=%s", tid, t.EmailList[0], err.Error())
		showMessage(w, "Error: couldn't send mail",
			`Something strange happened. Couldn't send
            out new password: `+err.Error(), tid, "")
		return
	}
	showMessage(w, "Check your mail!",
		"Mailed out your new password. Check your mail.", tid, "")
	TLog(context, t.ID, "", "reset password", r.RemoteAddr)
}

func editteamprompt(w http.ResponseWriter, r *http.Request) {
	session, tid := GetAndOrUpdateSession(w, r)
	if tid == "" {
		showMessage(w, "Error: not logged in", "You are not logged in", "", "/loginprompt")
		return
	}
	context := appengine.NewContext(r)
	t := TeamRecord{}
	key := datastore.NewKey(context, "Team", tid, 0, nil)
	err := datastore.Get(context, key, &t)
	if err != nil {
		context.Warningf("editteamprompt no read TID=%s ERR=%s", tid, err.Error())
		showMessage(w, "Error: couldn't read team data",
			`Something strange happened. Couldn't read the 
             team data: `+err.Error(), tid, "")
		return
	}
	t.EmailList = append(t.EmailList, "", "", "", "", "")
	token := session.actionToken("edit team")
	template.Must(template.New("").Parse(tEditTeamPrompt)).Execute(w, MapSI{
		"PageTitle":   "Edit Team Info",
		"Team":        tid,
		"EmailList":   append(t.EmailList, "", "", "", "")[:4],
		"Description": t.Description,
		"Token":       token,
		"TID":         tid,
		"AnnounceOK":  (t.AnnounceOK > 0),
	})
}

func editteam(w http.ResponseWriter, r *http.Request) {
	session, tid := GetAndOrUpdateSession(w, r)
	if tid == "" {
		showMessage(w, "Error: not logged in", "You are not logged in", "", "")
		return
	}
	token := session.actionToken("edit team")
	undoURL := "/editteamprompt"

	context := appengine.NewContext(r)
	if token != r.FormValue("token") {
		context.Errorf("editteam TOKEN TID=%s IP=%s", tid, r.RemoteAddr)
		showMessage(w, "Token didn't match",
			`Something strange happened. Are you sure you want
             to edit your team info?`, tid, undoURL)
		return
	}
	key := datastore.NewKey(context, "Team", tid, 0, nil)
	t := TeamRecord{}
	err := datastore.Get(context, key, &t)
	if err != nil {
		context.Warningf("edit(saver) no READ TID=%s ERR=%s", tid, err.Error())
		showMessage(w, "Error: couldn't read team data",
			`Something strange happened. Couldn't read the team data: `+
				err.Error(), tid, undoURL)
		return
	}
	emailList := []string{}
	for _, e := range r.Form["m"] {
		if len(strings.TrimSpace(e)) > 0 {
			emailList = append(emailList, e)
		}
	}
	if len(emailList) == 0 {
		showMessage(w, "Error: no email address specified",
			"Please specify at least one email address. (We need it to send the password to)", tid, undoURL)
		return
	}
	description := r.FormValue("description")
	if len(description) > 150 {
		description = description[:150]
	}
	announceok := 0
	if r.FormValue("announceok") != "" {
		announceok = 1
	}
	t.EmailList = emailList
	t.Description = description
	t.LastSeen = time.Now()
	t.AnnounceOK = announceok
	_, err = datastore.Put(context, key, &t)
	if err != nil {
		context.Warningf("Editteam (saver) no SAVE TID=%s ERR=%s", tid, err.Error())
		showMessage(w, "Error: Couldn't save team data",
			`Something strange happened. Couldn't save team data: `+
				err.Error(), tid, undoURL)
		return
	}
	showMessage(w, "Edited team info", "OK, edited.", tid, "")
}

func teamprofile(w http.ResponseWriter, r *http.Request) {
	_, tid := GetAndOrUpdateSession(w, r)
	context := appengine.NewContext(r)
	// TODO "Stud + Stones" shows up as "Stud+++Stones", and we're not
	// sure which are spaces and which are plusses.
	log.Printf("TEAM URL PATH %s", r.URL.Path)
	log.Printf("TEAM REQUESTURI %s", r.RequestURI)
	otherTeamID, err := url.QueryUnescape(r.URL.Path[len("/team/"):])
	logoutPrompt := ""
	context.Infof("TID         %s x", tid)
	context.Infof("OTHERTEAMID %s x", otherTeamID)
	if otherTeamID == tid {
		logoutPrompt = `<a href="/logout" class="btn">Logout</a>`
	}
	context.Infof("logoutPrompt %s", logoutPrompt)
	if otherTeamID == "" {
		w.WriteHeader(http.StatusNotFound)
		showMessage(w, "No such team",
			`Tried to view team "" but there is none such.`, tid, "")
		return
	}
	key := datastore.NewKey(context, "Team", otherTeamID, 0, nil)
	t := TeamRecord{}
	err = datastore.Get(context, key, &t)
	if err == datastore.ErrNoSuchEntity {
		w.WriteHeader(http.StatusNotFound)
		showMessage(w, "No such team",
			`Tried to view team "`+otherTeamID+`" but there is none such.`,
			tid, "")
		return
	}
	type badgedisplayinfo struct {
		Name        string
		Pretty      string
		Level       int
		Description string
	}
	var teamBadges = map[string]int{}
	dec := json.NewDecoder(strings.NewReader(t.Badges))
	dec.Decode(&teamBadges)
	var badgeDisplayInfo = []badgedisplayinfo{}
	for badge, level := range teamBadges {
		// Don't show legacy badges. Detectable since they're not in map:
		if badgeBling[badge].Pretty == "" {
			continue
		}
		badgeDisplayInfo = append(badgeDisplayInfo, badgedisplayinfo{
			Name:        badge,
			Pretty:      badgeBling[badge].Pretty,
			Level:       level,
			Description: badgeBling[badge].Description,
		})
	}
	g := getTeamGossip(context, otherTeamID)
	template.Must(template.New("").Parse(tTeamProfile)).Execute(w, MapSI{
		"PageTitle":    "Team Profile",
		"TID":          tid,
		"TeamID":       otherTeamID,
		"Description":  t.Description,
		"Gossip":       g,
		"Badges":       badgeDisplayInfo,
		"LogoutPrompt": logoutPrompt,
	})
}

// User wants to register a new team.  Let's clean up the input a lot.
// "Burninators", "Burninators ", "The Burninators", "Team Burninators"
// probably all refer to the same thing.
func normalizeNewTeamName(in string) (out string) {
	out = in
	if len(out) > 50 {
		out = out[:50]
	}
	out = strings.TrimSpace(out)
	if strings.HasPrefix(strings.ToLower(out), "the ") {
		out = strings.TrimSpace(out[3:])
	}
	// "Team Burninators" and "Burninators" are two ways of saying same thing
	if strings.HasPrefix(strings.ToLower(out), "team ") {
		out = strings.TrimSpace(out[4:])
	}
	out = strings.TrimSpace(out)
	if strings.HasPrefix(out, "/") {
		out = strings.Replace(out, "/", "-", 1)
	}
	if strings.HasPrefix(out, ".") {
		out = strings.Replace(out, ".", "-", 1)
	}
	return
}

// fetch team data from datastore, finding by id (nil if not found)
func getTeam(context appengine.Context, id string) *TeamRecord {
	if len(id) == 0 {
		id = "!empty!(^("
	}
	key := datastore.NewKey(context, "Team", id, 0, nil)
	t := TeamRecord{}
	err := datastore.Get(context, key, &t)
	if err != nil {
		context.Warningf("getteam no READ TID=%s ERR=%s", id, err.Error())
		return nil
	}
	return &t
}

// return a bunch of team names
func getTeamIDList(context appengine.Context) (l []string) {
	cache, err := memcache.Get(context, "Teams/IDList")
	if err == nil {
		return strings.Split(string(cache.Value), "\t")
	}
	if err == memcache.ErrCacheMiss {
		teams := make([]TeamRecord, 0, 50)
		q := datastore.NewQuery("Team").Order("-LastSeen")
		_, err = q.GetAll(context, &teams)
		if err != nil {
			context.Warningf("getteamidlist getall ERR=%s", err.Error())
		}
		for _, team := range teams {
			l = append(l, team.ID)
		}
		err = memcache.Set(context, &memcache.Item{
			Key:   "Teams/IDList",
			Value: []byte(strings.Join(l, "\t")),
		})
		if err != nil {
			context.Warningf("Teamlist no cache %s", err.Error())
		}
	} else {
		context.Warningf("Teamlist no cache read %s", err.Error())
	}
	return l
}

func getTeamIDsSimilarTo(context appengine.Context, tid string, slop int) (sims []string) {
	for _, id := range getTeamIDList(context) {
		if editDistance(tid, id) < ((len(tid)+len(id))/4)+slop {
			sims = append(sims, id)
		}
	}
	return
}

// Utility function to show a "message" screen.
func showMessage(w http.ResponseWriter, title string, message string, teamID string, backLink string) {
	template.Must(template.New("").Parse(tMessage)).Execute(w, map[string]string{
		"PageTitle": title,
		"Message":   message,
		"TID":       teamID,
		"GoBack":    backLink,
	})
}

func generateRandomPassword() (password string) {
	var (
		words = []string{
			"a", "A", "act", "alee", "all", "air", "ante", "art", "asta",
			"b", "B", "back", "bay", "been", "best", "big", "bug",
			"c", "cap", "cat", "city", "coy", "cup",
			"d", "D", "dab", "day", "den", "did", "dog", "dot", "dub",
			"e", "E", "each", "east", "ed", "egg", "epic", "era", "eve", "even",
			"f", "F", "fan", "far", "fez", "fin", "five", "fog", "fox", "fun",
			"g", "G", "gay", "get", "go", "h", "H", "hip", "hot", "how",
			"i", "ice", "idea", "ion", "iris", "is", "it", "item",
			"j", "J", "job", "joe", "joy", "june", "just",
			"k", "key", "kid", "kin", "king", "kit",
			"L", "law", "let", "lot", "love",
			"m", "M", "main", "mark", "max", "me", "met", "mix", "my",
			"n", "N", "name", "near", "need", "net", "news", "next", "now",
			"o", "oak", "odd", "of", "off", "on", "or", "own", "ox", "oz",
			"p", "pan", "park", "pat", "pen", "pop", "pro", "put", "q", "Q",
			"r", "R", "ram", "ran", "rap", "ray", "red", "rob", "roy", "run",
			"s", "saw", "say", "set", "ski", "sky", "spy", "sub", "such", "sum",
			"t", "T", "tax", "that", "they", "this", "time", "top", "tv",
			"u", "ufo", "up", "used", "v", "V", "van", "very",
			"w", "was", "way", "we", "web", "who", "win", "wit", "with", "x",
			"y", "yak", "year", "yes", "yet", "z", "zap", "zip", "zoo",
		}
		nums = []string{
			"2", "3", "4", "5", "6", "7", "8", "9", "17", "23", "26", "42",
		}
	)
	rand.Seed(int64(time.Now().Nanosecond()))
	for {
		password += words[rand.Intn(len(words))]
		if len(password) > 7 {
			return
		}
		password += nums[rand.Intn(len(nums))]
		if len(password) > 7 {
			return
		}
	}
	return
}
