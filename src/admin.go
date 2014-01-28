package octo

/*
 * UI for admins
 */

import (
	"appengine"
	"appengine/blobstore"
	"appengine/datastore"
	"appengine/taskqueue"
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type AdminAccountRecord struct {
	ID       string
	Password string
}

type ALogRecord struct {
	Created time.Time
	AID     string
	Verb    string
	Notes   string `datastore:",noindex"`
}

type ActFSRecord struct {
	B []byte
}

func adminmenu(w http.ResponseWriter, r *http.Request) {
	aid := checkAdminLogin(w, r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if aid == "" {
		fmt.Fprintf(w, `<form action="/admin/login" method=POST>
                        Admin account <input name="admin" type="text"><br>
                        Password <input name="passwd" type="password"><br>
                        <input type="submit" value="Login">
                        </form>`)
	} else {
		fmt.Fprintf(w, `Hello, %s!<ul>
          <li><a href="/admin/gossip">Gossip/Dashboard</a> / <a href="/admin/logs">Logs</a> / <a href="/admin/wtf">WTF Logs</a>
          <li><a href="/admin/teamspreadsheetv?arc=demo">&quot;Spreadsheet&quot; view</a>
          <li><a href="/admin/uploadprompt">Upload Activity or Arc</a>
          <li><a href="/admin/editadmin">Create/Edit Admin Accounts</a>
          <li><a href="/admin/maillist">List Emails of Folks w/Puzzle Open</a> in case of problem w/that puzzle
          </ul>`, html.EscapeString(aid))
	}

}

func adminlogin(w http.ResponseWriter, r *http.Request) {
	context := appengine.NewContext(r)
	admin := r.FormValue("admin")
	passwd := r.FormValue("passwd")
	c := http.Cookie{
		Name:   "aid",
		Value:  admin + ":" + passwd,
		MaxAge: SESSION_LIFETIME_S,
		Path:   "/",
	}
	http.SetCookie(w, &c)
	fmt.Fprintf(w, `It is done. <a href="/admin/">OK</a>`)
	if appengine.IsDevAppServer() {
		aa := AdminAccountRecord{
			ID:       admin,
			Password: passwd,
		}
		key := datastore.NewKey(context, "AdminAccount", admin, 0, nil)
		_, err := datastore.Put(context, key, &aa)
		if err == nil {
			ALog(context, admin, "fakeeditadmin", admin)
		}
	}
	ALog(context, admin, "login", r.RemoteAddr)
}

func admineditadmin(w http.ResponseWriter, r *http.Request) {
	aid := checkAdminLogin(w, r)
	context := appengine.NewContext(r)
	if aid == "" && !appengine.IsDevAppServer() {
		fmt.Fprintf(w, `Not logged in. <a href="/admin/">Go elsewhere</a>`)
		return
	}
	admin := r.FormValue("admin")
	passwd := r.FormValue("passwd")
	if admin != "" && passwd != "" {
		aa := AdminAccountRecord{
			ID:       admin,
			Password: passwd,
		}
		key := datastore.NewKey(context, "AdminAccount", admin, 0, nil)
		_, err := datastore.Put(context, key, &aa)
		if err == nil {
			ALog(context, aid, "editadmin", admin)
			fmt.Fprintf(w, `Saved. They can log in at 
                http://www.octothorpean.org/admin/login?admin=%s&amp;passwd=%s<br>
                <b><a href="/admin/">Back to admin menu</a></b><br><br>`,
				html.EscapeString(admin), html.EscapeString(passwd))
		} else {
			fmt.Fprintf(w, `Something went wrong saving the account: %s<br>`,
				err.Error())
		}
	}
	fmt.Fprintf(w, `<b>Create an Admin account (or edit an existing account)<br>
                    This clobbers the existing account, if any. Be careful</b>
                    <form method=POST>
                    ID: <input name="admin" type="text"> 
                    (just letters please)<br>
                    Passwd: <input name="passwd" type="text"> 
                    (just letters please)<br>
                    <input type="submit" value="Save"> 
                    <a href="/admin/">cancel</a>
                    </form>`)
}

func adminuploadprompt(w http.ResponseWriter, r *http.Request) {
	context := appengine.NewContext(r)
	aid := checkAdminLogin(w, r)
	if aid == "" {
		io.WriteString(w, "<p>You really need to log in. Sorry about that.")
		return
	}
	uploadURL, err := blobstore.UploadURL(context, "/admin/upload", nil)
	if err != nil {
		return
	}
	template.Must(template.New("").Parse(tUploadActPrompt)).Execute(w, MapSI{
		"PageTitle": "Upload",
		"UploadURL": uploadURL,
		"AID":       aid,
	})
}

func adminteamspreadsheet(w http.ResponseWriter, r *http.Request) {
	aid := checkAdminLogin(w, r)
	if aid == "" {
		io.WriteString(w, "<p>You really need to log in. Sorry about that.")
		return
	}
	context := appengine.NewContext(r)
	m := SummarizeLogs(context)
	arc := fetcharc(context, r.FormValue("arc"))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Team\t")
	for _, actID := range arc.Act {
		fmt.Fprintf(w, "%s Solve\t%s Hint\t", actID, actID)
	}
	fmt.Fprintf(w, "\n")
	for teamID, _ := range m {
		if teamID == "" {
			continue
		}
		fmt.Fprintf(w, "%s\t", teamID)
		for _, actID := range arc.Act {
			summ, ok := m[teamID][actID]
			if ok && summ.SolvedP {
				fmt.Fprintf(w, "%s\t%d\t", m[teamID][actID].SolveTime, m[teamID][actID].Hints)
			} else {
				fmt.Fprintf(w, "X\tX\t")
			}
		}
		fmt.Fprintf(w, "\n")
	}
}

func adminupload(w http.ResponseWriter, r *http.Request) {
	context := appengine.NewContext(r)
	blobmap, _, err := blobstore.ParseUpload(r)
	if err != nil {
		context.Errorf("UPLOAD couldn't parse upload %s", err)
		return
	}
	aid := checkAdminLogin(w, r)
	if aid == "" {
		io.WriteString(w, "<p>You really need to log in. Sorry about that.")
		return
	}
	ALog(context, aid, "upload", "???")
	file := blobmap["file"]
	if len(file) == 0 {
		context.Errorf("no file uploaded")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	for _, b := range blobmap["file"] {
		t := taskqueue.NewPOSTTask("/admin/digestupload",
			map[string][]string{"blobkey": {string(b.BlobKey)}})
		if _, err := taskqueue.Add(context, t, "digest"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func adminuploadintera(w http.ResponseWriter, r *http.Request) {
	context := appengine.NewContext(r)
	f, _, err := r.FormFile("file")
	if err != nil {
		context.Errorf("ERR INTERA OPEN %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	var b bytes.Buffer
	io.Copy(&b, f)
	var afsr ActFSRecord
	afsr.B = b.Bytes()
	aid := checkAdminLogin(w, r)
	if aid == "" {
		io.WriteString(w, "<p>You really need to log in. Sorry about that.")
		return
	}
	key := datastore.NewKey(context, "ActFSRecord", "!/INTERA.txt", 0, nil)
	_, err = datastore.Put(context, key, &afsr)
	interacache = map[string]ICache{}
	arccache = map[string]ArcCache{}
	if err == nil {
		ALog(context, aid, "uploadintera", r.RemoteAddr)
	}
}

// Fill in activityrecord "in place" based on contents of puz.txt file
func ReadPuzTxt(context appengine.Context, f *zip.File, act *ActivityRecord) {
	rc, err := f.Open()
	if err != nil {
		context.Errorf("ERR READPUZTXT OPEN: %s", err)
		return
	}
	lineReader := bufio.NewReader(rc)
	for {
		l, isPrefix, err := lineReader.ReadLine()
		line := string(l)
		if isPrefix {
			context.Errorf("ERR READPUZTXT TRUNCATED LONG LINE %s", line)
		}
		if err == io.EOF {
			break
		}
		if strings.HasPrefix(line, "NICKNAME") {
			act.Nickname = scrunch(line[8:])
		}
		if strings.HasPrefix(line, "TITLE") {
			act.Title = strings.TrimSpace(line[5:])
		}
		if strings.HasPrefix(line, "GCNOTE") {
			act.GCNote = line[6:]
		}
		if strings.HasPrefix(line, "SOLUTION") {
			act.Solutions = append(act.Solutions, scrunch(line[8:]))
		}
		if strings.HasPrefix(line, "PARTIAL") {
			act.Partials = append(act.Partials, strings.TrimSpace(line[7:]))
		}
		if strings.HasPrefix(line, "HINT") {
			act.Hints = append(act.Hints, strings.TrimSpace(line[4:]))
		}
		if strings.HasPrefix(line, "TAGS") {
			rawtags := strings.Split(strings.TrimSpace(line[4:]), ",")
			var already = map[string]bool{}
			var implied = map[string][]string{
				"5bit":        {"numeric", "code"},
				"7segment":    {"electronic"},
				"a1":          {"numeric", "code"},
				"anagram":     {"word"},
				"braille":     {"code"},
				"crossword":   {"word"},
				"dropquote":   {"word"},
				"fillomino":   {"nikoli", "logic"},
				"hashi":       {"nikoli", "logic"},
				"indexing":    {"numeric"},
				"masyu":       {"nikoli", "logic"},
				"morse":       {"code"},
				"nonogram":    {"conceptis", "logic"},
				"phonespell":  {"electronic"},
				"pigpen":      {"code"},
				"riddles":     {"popculture"},
				"semaphore":   {"code", "flags"},
				"shikaku":     {"nikoli", "logic"},
				"slitherlink": {"nikoli", "logic"},
				"sudoku":      {"logic"},
				"tentaishow":  {"nikoli", "logic"},
				"tv":          {"popculture"},
				"wordsearch":  {"word"},
			}
			for _, tag := range act.Tags {
				already[tag] = true
			}
			for _, rawtag := range rawtags {
				tag := scrunch(rawtag)
				if tag == "" {
					continue
				}
				if already[tag] {
					continue
				}
				already[tag] = true
				for _, subtag := range implied[tag] {
					if already[subtag] {
						continue
					}
					already[subtag] = true
				}
			}
			if !already["act"] {
				act.Tags = append(act.Tags, "act")
			}
			already["act"] = true
			act.Tags = []string{}
			for tag, _ := range already {
				act.Tags = append(act.Tags, tag)
			}
		}
	}
	return
}

// probably run in a queued task REMIND or a backend?
func digestupload(w http.ResponseWriter, r *http.Request) {
	context := appengine.NewContext(r)
	blobkey := appengine.BlobKey(r.FormValue("blobkey"))
	stat, err := blobstore.Stat(context, blobkey)
	ALog(context, "???", "digest start", "???")
	if err != nil {
		context.Errorf("ERR STAT: %s", err)
		return
	}
	blobreader := blobstore.NewReader(context, blobkey)
	zipreader, err := zip.NewReader(blobreader, stat.Size)
	if err != nil {
		context.Errorf("ERR NEWREADER: %s", err)
		return
	}
	act := new(ActivityRecord)
	topDir := ""
	found := false
	for _, f := range zipreader.File {
		if strings.ToLower(path.Base(f.Name)) == "puz.txt" {
			ReadPuzTxt(context, f, act)
			found = true
			topDir, _ = path.Split(f.Name)
		}
		if strings.ToLower(path.Base(f.Name)) == "index.html" {
			rc, err := f.Open()
			if err != nil {
				context.Errorf("ERR index.html open %s", err)
				continue
			}
			reader := bufio.NewReader(rc)
			act.Guts, err = reader.ReadBytes('\000')
			if err != nil && err != io.EOF {
				context.Errorf("ERR reading index.html %s", err)
				continue
			}
		}
	}
	if !found {
		context.Errorf("ERR No puz.txt file in zip file, don't know what to do")
		return
	}
	key := datastore.NewKey(context, "Activity", act.Nickname, 0, nil)
	_, err = datastore.Put(context, key, act)
	if err != nil {
		context.Errorf("Activity save failed ACT %s ERR %s",
			act.Nickname, err.Error())
	}
	errCount := 0
	for _, f := range zipreader.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		if strings.ToLower(path.Base(f.Name)) == "puz.txt" {
			continue
		}
		if strings.ToLower(path.Base(f.Name)) == "index.html" {
			continue
		}
		path := act.Nickname + "/" + strings.Replace(f.Name, topDir, "", 1)
		rc, err := f.Open()
		if err != nil {
			context.Errorf("ERR %s OPEN %s", f.Name, err)
			errCount++
			continue
		}
		reader := bufio.NewReaderSize(rc, int(f.UncompressedSize))
		var afsr ActFSRecord
		var b bytes.Buffer
		_, err = io.Copy(&b, reader)
		if err != nil {
			context.Errorf("ERR %s READ %s", f.Name, err)
			errCount++
			continue
		}
		afsr.B = b.Bytes()
		key := datastore.NewKey(context, "ActFSRecord", path, 0, nil)
		_, err = datastore.Put(context, key, &afsr)
		if err != nil {
			context.Errorf("ERR %s WRITE %s", path, err)
			errCount++
			continue
		}
	}
	if errCount == 0 {
		blobstore.Delete(context, blobkey)
	}
	ALog(context, "???", "digest finish", act.Nickname)
}

func admineditactivity(w http.ResponseWriter, r *http.Request) {
	context := appengine.NewContext(r)
	aid := checkAdminLogin(w, r)
	if aid == "" {
		aid = r.FormValue("admin")
		if aid == "" {
			return
		}
		key := datastore.NewKey(context, "AdminAccount", aid, 0, nil)
		s := AdminAccountRecord{}
		err := datastore.Get(context, key, &s)
		if err != nil {
			fmt.Fprintf(w, "Couldn't get admin account: ", err.Error())
			return
		}
		if s.Password != r.FormValue("passwd") {
			fmt.Fprintf(w, `Password did not match`)
			return
		}
	}
	nickname := scrunch(r.FormValue("nickname"))
	title := r.FormValue("title")
	gcnote := r.FormValue("gcnote")
	url := r.FormValue("url")
	sols := []string{}
	for i, s := range r.Form["sols"] {
		if i == 0 {
			continue
		}
		if strings.TrimSpace(s) != "" {
			sols = append(sols, s)
		}
	}
	// TODO this is silly. Need a func to remove blank strings from slice
	partials := []string{}
	for i, p := range r.Form["part"] {
		if i == 0 {
			continue
		}
		if strings.TrimSpace(p) != "" {
			partials = append(partials, p)
		}
	}
	hints := []string{}
	for i, h := range r.Form["hint"] {
		if i == 0 {
			continue
		}
		if strings.TrimSpace(h) != "" {
			hints = append(hints, h)
		}
	}
	guts := r.FormValue("guts")
	status := ""

	if r.FormValue("verb") == "load" && nickname != "" {
		title = "something happened or did not happen"
		key := datastore.NewKey(context, "Activity", nickname, 0, nil)
		a := ActivityRecord{}
		err := datastore.Get(context, key, &a)
		if err == nil {
			title = a.Title
			gcnote = a.GCNote
			url = a.URL
			guts = string(a.Guts)
			sols = a.Solutions
			partials = a.Partials
			// hey would this be easier if I just used this record 
			// instead of separate local vars in the first place?
			hints = a.Hints
		} else {
			status = "Load failed! " + err.Error()
		}
	} else if nickname != "" {
		ALog(context, aid, "editact", nickname)
		key := datastore.NewKey(context, "Activity", nickname, 0, nil)
		a := ActivityRecord{
			Nickname:  nickname,
			Title:     title,
			GCNote:    gcnote,
			URL:       url,
			Guts:      []byte(guts),
			Solutions: sols,
			Partials:  partials,
			Hints:     hints,
		}
		_, err := datastore.Put(context, key, &a)
		if err != nil {
			context.Warningf("Activity save failed AD %s ACT %s ERR %s",
				aid, nickname, err.Error())
		}
	}

	template.Must(template.New("").Parse(tAdminEditActivity)).Execute(w, MapSI{
		"PageTitle": "Edit Activity: " + nickname,
		"Nickname":  nickname,
		"Title":     title,
		"GCNote":    gcnote,
		"URL":       url,
		"Guts":      guts,
		"Solutions": append(sols, "", "", ""),
		"Partials":  append(partials, "", "", "", ""),
		"Hints":     append(hints, "", ""),
		"Status":    status,
	})
}

func checkAdminLogin(w http.ResponseWriter, r *http.Request) (adminLogin string) {
	adminLogin = ""
	cv := ""
	for _, c := range r.Cookies() {
		if c.Name == "aid" {
			cv = c.Value
			break
		}
	}
	loginAndPasswd := strings.SplitN(cv, ":", 2)
	if len(loginAndPasswd) != 2 {
		return
	}
	if len(loginAndPasswd[0]) == 0 {
		return
	}
	context := appengine.NewContext(r)
	key := datastore.NewKey(context, "AdminAccount", loginAndPasswd[0], 0, nil)
	s := AdminAccountRecord{}
	err := datastore.Get(context, key, &s)
	if err != nil {
		fmt.Fprintf(w, `Something went wrong looking up account "%s" %s<br>`,
			html.EscapeString(loginAndPasswd[0]), err.Error())
		return
	}
	if s.Password != loginAndPasswd[1] {
		fmt.Fprintf(w, `Password "%s" did not match<br>
                        <a href="/admin/login">Log in</a> or something?<br>`,
			html.EscapeString(loginAndPasswd[1]))
		return
	}
	return loginAndPasswd[0]
}

func ALog(context appengine.Context, aid string, verb string, notes string) error {
	a := ALogRecord{
		Created: time.Now(),
		AID:     aid,
		Verb:    verb,
		Notes:   notes,
	}
	_, err := datastore.Put(context, datastore.NewIncompleteKey(context, "ALog", nil), &a)
	if err != nil {
		context.Errorf("Error writing ALog A %s V %s N %s ERR %s",
			aid, verb, notes, err.Error())
	}
	return err
}

func adminlogs(w http.ResponseWriter, r *http.Request) {
	aid := checkAdminLogin(w, r)
	if aid == "" {
		fmt.Fprintf(w, `alert("Not logged in!");`)
		return
	}
	context := appengine.NewContext(r)
	team := r.FormValue("team")
	act := r.FormValue("act")
	verb := r.FormValue("verb")
	filters := []string{}
	limit := 500
	tableheader := "<tr>"
	if team == "" {
		tableheader = tableheader + "<th>Team"
	} else {
		filters = append(filters, "team = "+team)
	}
	if act == "" {
		tableheader = tableheader + "<th>Act"
	} else {
		filters = append(filters, "act = "+act)
		limit = limit * 10
	}
	if verb == "" {
		tableheader = tableheader + "<th>Verb"
	} else {
		filters = append(filters, "verb = "+verb)
	}
	tableheader = tableheader + "<th>Guess <th>#/Notes <th>Created"
	rows := []template.HTML{}
	q := datastore.NewQuery("TLog").Order("-Created").Limit(limit)
	// filter on team or verb, but not both. (why not both? I'm too miserly
	// to create another index for such rarely-used queries)
	if team != "" {
		q = q.Filter("TeamID=", team)
	} else if verb != "" {
		q = q.Filter("Verb=", verb)
	}
	for iter := q.Run(context); ; {
		var tlr TLogRecord
		_, err := iter.Next(&tlr)
		if err != nil {
			break
		}
		row := "<tr>"
		if team == "" {
			row = row + "<td>" + template.HTMLEscapeString(tlr.TeamID)
		} else {
			if tlr.TeamID != team {
				continue
			}
		}
		if act == "" {
			row = row + "<td>" + tlr.ActID
		} else {
			if tlr.ActID != act {
				continue
			}
		}
		if verb == "" {
			row = row + "<td>" + tlr.Verb
		} else {
			if tlr.Verb != verb {
				continue
			}
		}
		note := ""
		if tlr.Hint > 0 {
			note = fmt.Sprintf("%d %s", tlr.Hint, tlr.Notes)
		} else {
			note = tlr.Notes
		}
		row = row + fmt.Sprintf("<td>%s <td>%s <td><span class=\"date\">%s</span>",
			tlr.Guess, note, tlr.Created)
		rows = append(rows, template.HTML(row))
	}
	template.Must(template.New("").Parse(tAdminLogs)).Execute(w, MapSI{
		"PageTitle":   "Admin / Logs",
		"Filters":     filters,
		"TableHeader": template.HTML(tableheader),
		"Rows":        rows,
	})
}

func admingossip(w http.ResponseWriter, r *http.Request) {
	template.Must(template.New("").Parse(tAdminGossip)).Execute(w, struct {
		PageTitle string
	}{
		PageTitle: "Admin / Gossip",
	})
}

func admingossipjson(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript")
	aid := checkAdminLogin(w, r)
	if aid == "" {
		fmt.Fprintf(w, `alert("Not logged in!");`)
		return
	}
	context := appengine.NewContext(r)
	fetched := fetchGossip(context)
	l := []tidbit{}
	for _, tlr := range fetched {
		t := fmt.Sprintf(`Team <a href="/team/%s">%s</a>`,
			html.EscapeString(url.QueryEscape(url.QueryEscape(tlr.TeamID))),
			html.EscapeString(tlr.TeamID))
		a := fmt.Sprintf(`<a href="/a/%s/">%s</a>`,
			html.EscapeString(tlr.ActID), html.EscapeString(tlr.ActID))
		v := tlr.Verb
		g := html.EscapeString(tlr.Guess)
		n := html.EscapeString(tlr.Notes)
		l = append(l, tidbit{
			T: tlr.Created.Unix() * 1000,
			M: fmt.Sprintf(`%s %s %s %s %s`, t, v, a, g, n),
		})
	}
	spewjsonp(w, r, MapSI{"gossip": l})
}

func adminwtflogs(w http.ResponseWriter, r *http.Request) {
	DEPTH := 50    // if one item has this many hits, we're done
	BREADTH := 200 // if we've seen this many items, we're done
	var count = map[string]int{}
	context := appengine.NewContext(r)
	q := datastore.NewQuery("TLog").Order("-Created").Filter("Verb=", "wtfguess")
	for iter := q.Run(context); ; {
		var tlr TLogRecord
		_, err := iter.Next(&tlr)
		if err == datastore.Done {
			break
		}
		if err != nil {
			context.Warningf("TeamGossip iter ERR %s", err.Error())
			break
		}
		mapkey := tlr.ActID + ":" + tlr.Guess
		count[mapkey] = count[mapkey] + 1
		if count[mapkey] >= DEPTH {
			break
		}
		if len(count) >= BREADTH {
			break
		}
	}
	outs := []string{}
	for i := DEPTH; i > 1; i-- { // if we were prostyle, we'd sort. 
		for key, value := range count {
			if value != i {
				continue
			}
			outs = append(outs, fmt.Sprintf("%s %d", key, value))
		}
	}
	template.Must(template.New("").Parse(tAdminWTFLogs)).Execute(w, MapSI{
		"PageTitle": "WTF",
		"Counts":    outs,
	})
}

func adminmaillist(w http.ResponseWriter, r *http.Request) {
	aid := checkAdminLogin(w, r)
	if aid == "" {
		fmt.Fprintf(w, `alert("Not logged in!");`)
		return
	}
	actID := r.FormValue("act")
	if actID == "" {
		actID = "ui"
	}

	context := appengine.NewContext(r)

	q := datastore.NewQuery("TAState").Filter("ActID=", actID)

	teamIDs := []string{}
	addresses := []string{}
	var teamKeys []*datastore.Key

	for iter := q.Run(context); ; {
		var tas TAStateRecord
		_, err := iter.Next(&tas)
		if err != nil {
			break
		}
		if tas.SolvedP {
			continue
		}
		teamIDs = append(teamIDs, tas.TeamID)
		teamKeys = append(teamKeys, datastore.NewKey(context, "Team", tas.TeamID, 0, nil))
	}

	teams := make([]TeamRecord, len(teamKeys))

	datastore.GetMulti(context, teamKeys, teams)

	for _, team := range teams {
		for _, a := range team.EmailList {
			addresses = append(addresses, a)
		}
	}

	template.Must(template.New("").Parse(tAdminMailList)).Execute(w, MapSI{
		"PageTitle": "MailList",
		"Act":       actID,
		"Teams":     teamIDs,
		"Addresses": addresses,
	})
}

// Dump tab-separated-values 
func admindumpteamlogs(w http.ResponseWriter, r *http.Request) {
	aid := checkAdminLogin(w, r)
	if aid == "" {
		io.WriteString(w, "<p>You really need to log in. Sorry about that.")
		return
	}
	context := appengine.NewContext(r)

	// Grabbing the million most recent logs isn't so useful; for our
	// measuring, we want to know how _teams_ make their way. So grab logs
	// from recent _teams_. (If an old team has been active recently, that's
	// interesting for some analyses... but not for what we're doing _here_.)
	recentTeamIDs := []string{}
	q := datastore.NewQuery("Team").Order("-Created").Limit(100)
	for iter := q.Run(context); ; {
		var tr TeamRecord
		_, err := iter.Next(&tr)
		if err != nil {
			break
		}
		recentTeamIDs = append(recentTeamIDs, tr.ID)
	}

	records := [][]string{}

	for _, recentTeamID := range recentTeamIDs {
		q = datastore.NewQuery("TLog").Filter("TeamID=", recentTeamID)
		for iter := q.Run(context); ; {
			var tlr TLogRecord
			_, err := iter.Next(&tlr)
			if err != nil {
				break
			}
			asStringArray := []string{
				fmt.Sprintf("%d", tlr.Created.Unix()),
				tlr.TeamID,
				tlr.ActID,
				tlr.Verb,
				tlr.Guess,
				fmt.Sprintf("%d", tlr.Hint),
				tlr.Notes,
			}
			records = append(records, asStringArray)
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	tsvWriter := csv.NewWriter(w)
	tsvWriter.Comma = '\t'
	tsvWriter.WriteAll(records)
}

// yipe team RedNation hit a highly-visible double-counting bug:
// their conspiracy badge got to level 2
func admincleanteam(w http.ResponseWriter, r *http.Request) {
	aid := checkAdminLogin(w, r)
	if aid == "" {
		io.WriteString(w, "<p>You really need to log in. Sorry about that.")
		return
	}
	context := appengine.NewContext(r)
	teamID := "RedNation"
	key := datastore.NewKey(context, "Team", teamID, 0, nil)
	tr := TeamRecord{}
	datastore.Get(context, key, &tr)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `Before:<br>`)
	fmt.Fprintf(w, ` Badges:%s<br>`, tr.Badges)
	fmt.Fprintf(w, ` Tags:%s<br>`, tr.Tags)
	replfrom := `"conspiracy":2`
	replto := `"conspiracy":1`
	tr.Badges = strings.Replace(tr.Badges, replfrom, replto, 1)
	tr.Tags = strings.Replace(tr.Tags, replfrom, replto, 1)
	fmt.Fprintf(w, `After:<br>`)
	fmt.Fprintf(w, ` Badges:%s<br>`, tr.Badges)
	fmt.Fprintf(w, ` Tags:%s<br>`, tr.Tags)
	datastore.Put(context, key, &tr)
}

// yipe team RedNation hit a highly-visible double-counting bug:
// their conspiracy badge got to level 2
func admineditteam(w http.ResponseWriter, r *http.Request) {
	aid := checkAdminLogin(w, r)
	if aid == "" {
		io.WriteString(w, "<p>You really need to log in. Sorry about that.")
		return
	}
	context := appengine.NewContext(r)
	enteredTeam := r.FormValue("enteredteam") 
	tr := TeamRecord{}
    var err error
	if enteredTeam != "" {
		key := datastore.NewKey(context, "Team", enteredTeam, 0, nil)
		err = datastore.Get(context, key, &tr)
		if err != nil {
			context.Warningf("editteamprompt no read TID=%s ERR=%s", enteredTeam, err.Error())
		}
	}
	if (r.FormValue("yarly") == "on") && (tr.ID != "") {
		key := datastore.NewKey(context, "Team", tr.ID, 0, nil)
		tr.EmailList = r.Form["m"]
		tr.AnnounceOK = 0
		if r.FormValue("announceok") == "on" {
			tr.AnnounceOK = 1
		}
		tr.Tags = r.FormValue("tags")
		tr.Badges = r.FormValue("badges")
		ALog(context, aid, "editteam", tr.ID)
		TLog(context, tr.ID, "", "adminedit", aid)
		if r.FormValue("logpcard") == "on" {
			TLog(context, tr.ID, "ui", "badge", "postcard")
		}
		_, err = datastore.Put(context, key, &tr)
	}
	template.Must(template.New("").Parse(tAdminEditTeam)).Execute(w, MapSI{
		"PageTitle": "editing team",
		"EnteredTeam": enteredTeam,
		"Error": err,
		"Team": tr.ID,
		"EmailList": tr.EmailList,
		"AnnounceOK": tr.AnnounceOK,
		"Tags": tr.Tags,
		"Badges": tr.Badges,
	})
}
