package octo

/**
 * Storing team logs and other team state.
 */

import (
	"appengine"
	"appengine/datastore"
	//    "log"
	"time"
)

// Basic info about each team.  There is one of these records per team.
type TeamRecord struct {
	ID          string
	Created     time.Time
	LastSeen    time.Time
	EmailList   []string
	Password    string `datastore:",noindex"`
	Description string `datastore:",noindex"`
	Tags        string `datastore:",noindex"` // JSON-encoded map of "experience points"
	Badges      string `datastore:",noindex"` // JSON-encoded achievements unlocked
	AnnounceOK  int
}

// How is this team doing with this Activity? There is one of these records
// for each team/activity pair (or no such record if team hasn't interacted
// w/this activity yet).
type TAStateRecord struct {
	TeamID  string
	ActID   string
	SolvedP bool // Did they solve it?
	Hints   int  // How many hints did they "buy"?
}

// What has this team done? A team will create many of these logs as it 
// is created, logs in, makes guesses, sovlves puzzles, asks for hints...
type TLogRecord struct {
	Created time.Time
	TeamID  string
	ActID   string
	Verb    string // "reg", "login", "guess", "hint", "resetpsswd", ...
	Guess   string `datastore:",noindex"`
	Hint    int    `datastore:",noindex"`
	Notes   string `datastore:",noindex"`
}

func TLog(context appengine.Context, teamID string, actID string, verb string, notes string) error {
	t := TLogRecord{
		Created: time.Now(),
		TeamID:  teamID,
		ActID:   actID,
		Verb:    verb,
		Notes:   notes,
	}
	_, err := datastore.Put(context, datastore.NewIncompleteKey(context, "TLog", nil), &t)
	// update team.LastSeen
	if err == nil && verb == "login" {
		datastore.RunInTransaction(context, func(c appengine.Context) error {
			key := datastore.NewKey(context, "Team", teamID, 0, nil)
			tr := TeamRecord{}
			err := datastore.Get(context, key, &tr)
			if err == nil {
				tr.LastSeen = time.Now()
				_, err = datastore.Put(context, key, &tr)
			}
			return err
		}, nil)
	}

	if err != nil {
		context.Errorf("Error writing TLog T %s A %s V %s N %s ERR %s",
			teamID, actID, verb, notes, err.Error())
	}
	return err
}

func TLogGuess(context appengine.Context, teamID string, actID string, verb string, guess string) error {
	t := TLogRecord{
		Created: time.Now(),
		TeamID:  teamID,
		ActID:   actID,
		Verb:    verb,
		Guess:   guess,
	}
	// TODO is this hack useful? I'm seeing "solves" that happen before the
	// relevant "guess".  So let's add a moment to the solve time:
	if verb == "solve" {
		t.Created = time.Now().Add(time.Millisecond)
	}
	_, err := datastore.Put(context, datastore.NewIncompleteKey(context, "TLog", nil), &t)
	// update team.LastSeen. But (hack) not if team solved, because in
	// that case, we're about to spend a lot of time writing other things
	// to datastore. And team.LastSeen doesn't need to be super-accurate
	if err == nil && verb != "solve" {
		datastore.RunInTransaction(context, func(c appengine.Context) error {
			key := datastore.NewKey(context, "Team", teamID, 0, nil)
			tr := TeamRecord{}
			err := datastore.Get(context, key, &tr)
			if err == nil {
				tr.LastSeen = time.Now()
				_, err = datastore.Put(context, key, &tr)
			}
			return err
		}, nil)
	}
	if err != nil {
		context.Errorf("Error writing TLog T %s A %s V %s G %s ERR %s",
			teamID, actID, verb, guess, err.Error())
	}
	return err
}

func TLogHint(context appengine.Context, teamID string, actID string, hint int) error {
	t := TLogRecord{
		Created: time.Now(),
		TeamID:  teamID,
		ActID:   actID,
		Verb:    "hint",
		Hint:    hint,
	}
	_, err := datastore.Put(context, datastore.NewIncompleteKey(context, "TLog", nil), &t)
	if err != nil {
		context.Errorf("Error writing TLog T %s A %s V hint H %d ERR %s",
			teamID, actID, hint, err.Error())
	}
	return err
}

// Is this team guessing via a dictionary attack? Let's count their recent
// guesses.
func TLogCountRecentGuesses(context appengine.Context, teamId string) int {
	q := datastore.NewQuery("TLog").Order("-Created").Filter("TeamID=", teamId).Filter("Created >", time.Now().Add(time.Minute*time.Duration(-5))).KeysOnly()
	count, err := q.Count(context)
	if err != nil {
		context.Warningf("CountGuesses GET whoops ERR=%s", err.Error())
	}
	return count
}

func CleanupTeamLogs(context appengine.Context) {
	// TODO
}

// Given a team and a set of acts, determine which acts the team
// has not yet unlocked.
func GetLockedActs(context appengine.Context, tid string, actIDs []string) []string {
	keys := make([]*datastore.Key, len(actIDs))
	tass := make([]TAStateRecord, len(actIDs))
	for ix, actID := range actIDs {
		keys[ix] = datastore.NewKey(context, "TAState", actID+":"+tid, 0, nil)
	}
	datastore.GetMulti(context, keys, tass)
	var retval []string
	for ix, tas := range tass {
		if tas.TeamID == "" {
			retval = append(retval, actIDs[ix])
		}
	}
	return retval
}

// If team has not already unlocked this act, then unlock it
func UnlockAct(context appengine.Context, tid string, actID string) {
	tas := TAStateRecord{}
	key := datastore.NewKey(context, "TAState", actID+":"+tid, 0, nil)
	datastore.Get(context, key, &tas)
	if tas.TeamID == "" {
		tas.TeamID = tid
		tas.ActID = actID
		tas.SolvedP = false
		tas.Hints = 0
	}
	datastore.Put(context, key, &tas)
}

// Sometimes, GC wants a "spreadsheety" view of teams instead of a "log/diary"
// view.
type SummaryElement struct {
	SolvedP   bool
	SolveTime time.Time
	Hints     int
}

func SummarizeLogs(context appengine.Context) (t map[string](map[string]*SummaryElement)) {
	t = map[string](map[string]*SummaryElement){}
	// query the logs: we are interested in solves and hint-takings.
	q := datastore.NewQuery("TLog").
		// would be nice to filter for Verb IN {"hint", "solve"}
		Filter("Created >", time.Now().Add(time.Hour*time.Duration(-999))).
		Order("-Created")
	for iter := q.Run(context); ; {
		var tlr TLogRecord
		_, err := iter.Next(&tlr)
		if err != nil {
			break
		}
		if len(t[tlr.TeamID]) == 0 {
			t[tlr.TeamID] = map[string]*SummaryElement{}
		}
		_, ok := t[tlr.TeamID][tlr.ActID]
		if !ok {
			t[tlr.TeamID][tlr.ActID] = new(SummaryElement)
		}
		if tlr.Verb == "hint" {
			if t[tlr.TeamID][tlr.ActID].Hints < tlr.Hint {
				t[tlr.TeamID][tlr.ActID].Hints = tlr.Hint
			}
		}
		if tlr.Verb == "solve" {
			t[tlr.TeamID][tlr.ActID].SolvedP = true
			t[tlr.TeamID][tlr.ActID].SolveTime = tlr.Created
		}
	}
	return
}
