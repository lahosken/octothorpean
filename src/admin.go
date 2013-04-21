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
	"fmt"
	"html"
	"io"
	"net/http"
	"path"
	"strings"
	"text/template"
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
          <li><a href="/admin/teamspreadsheet.tsv?arc=mendy">&quot;Spreadsheet&quot; view</a>
          <li><a href="/admin/uploadprompt">Upload Activity or Arc</a>
          <li><a href="/admin/editadmin">Create/Edit Admin Accounts</a>
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
		context.Infof("HEJ")
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
	uploadURL, err := blobstore.UploadURL(context, "/admin/upload", nil)
	if err != nil {
		return
	}
	template.Must(template.New("").Parse(tUploadActPrompt)).Execute(w, MapSI {
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
		if teamID == "" { continue }
		fmt.Fprintf(w, "%s\t",  teamID)
		for _, actID := range arc.Act {
			summ, ok := m[teamID][actID]
			if ok && summ.SolvedP {
				fmt.Fprintf(w, "%s\t%d\t",  m[teamID][actID].SolveTime, m[teamID][actID].Hints)
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
			var already = map[string] bool{}
			var implied = map[string] []string {
				"5bit": { "numeric", "code" },
				"7segment": { "electronic" },
				"a1" : { "numeric", "code" },
				"anagram": { "word" },
				"braille": { "code" },
				"crossword": { "word" },
				"dropquote": { "word" },
				"fillomino": { "nikoli", "logic" },
				"hashi": { "nikoli", "logic" },
				"indexing": { "numeric" }, 
				"masyu": { "nikoli", "logic" },
				"morse": { "code" },
				"nonogram": { "conceptis", "logic" },
				"phonespell": { "electronic" },
				"pigpen" : { "code" },
				"riddles": { "popculture" },
				"semaphore": { "code", "flags" },
				"shikaku": { "nikoli", "logic" },
				"slitherlink": { "nikoli", "logic" },
				"sudoku": { "logic" },
				"tentaishow": { "nikoli", "logic" },
				"tv": { "popculture" },
				"wordsearch": { "word" },
			}
			for _, tag := range act.Tags {
				already[tag] = true
			}
			for _, rawtag := range rawtags {
				tag := scrunch(rawtag)
				if tag == "" { continue }
				if already[tag] { continue }
				already[tag] = true
				for _, subtag := range implied[tag] {
					if already[subtag] { continue }
					already[subtag] = true
				}
			}
			if !already["act"] { act.Tags = append(act.Tags, "act") }
			already["act"] = true
			act.Tags = []string{}
			for tag, _ := range(already) {
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
