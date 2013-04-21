package octo

/**
 * session: give user's browser a cookie to keep track of their team
 *   We give the user a "sid" (session ID) cookie w/random value like "12345"
 *   We have a persistent store of session ID -> team ID. We can remember 
 *   that session 12345 is logged in as "mighty-atom".
 */

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"errors"
	"fmt"
	"hash/adler32"
	"math/rand"
	"net/http"
	"time"
)

const (
	SESSION_LIFETIME_S = 2097152 // nice round number, ~24 days in seconds
)

var (
	SessionAlreadyExists = errors.New("Keep going")
)

type Session struct {
	ID string
}

type SessionRecord struct {
	Created time.Time
	TeamID  string
}

// Call this at the start of your response handler. It returns the session
// and the team ID. (Returned team ID is "" means user is not logged in.)
// + Retrieves the session cookie.
// + Sets that cookie if not found. (That's why you need to call this function
//   before your handler writes any text to the ResponseWriter.)
// + Looks up session to see if team associated with it
// + Returns session and team IDs
func GetAndOrUpdateSession(w http.ResponseWriter, r *http.Request) (session *Session, teamID string) {
	session = new(Session)
	c, err := r.Cookie("sid")
	if err == nil {
		session.ID = c.Value
	}
	context := appengine.NewContext(r)
	if session.ID == "" { // new user. let's give them a session
		sid := newSessionRecordID(context)
		c := http.Cookie{
			Name:   "sid",
			Value:  sid,
			Path:   "/",
			MaxAge: SESSION_LIFETIME_S,
		}
		http.SetCookie(w, &c)
		s := SessionRecord{
			Created: time.Now(),
			TeamID:  "",
		}
		key := datastore.NewKey(context, "Session", sid, 0, nil)
		_, err := datastore.Put(context, key, &s)
		if err != nil {
			context.Warningf("Put new sess. SID=%s ERR= %s", sid, err.Error())
		}
		session.ID = sid
	}

	// Get teamID from datastore-backed cache (if known)
	cache, err := memcache.Get(context, "Session/"+session.ID)
	if err == nil {
		teamID = string(cache.Value)
		return
	}
	key := datastore.NewKey(context, "Session", session.ID, 0, nil)
	s := SessionRecord{}
	err = datastore.Get(context, key, &s)
	if err != nil {
		context.Errorf("Get sess. SID=%s ERR= %s", session.ID, err.Error())
		return
	}
	teamID = s.TeamID
	// This session->team mapping wasn't cached. Cache it for next time.
	err = memcache.Set(context, &memcache.Item{
		Key:   "Session/" + session.ID,
		Value: []byte(teamID),
	})
	if err != nil {
		context.Warningf("Cache sess. SID=%s ERR= %s", session.ID, err.Error())
	}
	return
}

// "claim" a new session record.
func newSessionRecordID(context appengine.Context) (id string) {
	rand.Seed(int64(time.Now().Nanosecond()))
	s := SessionRecord{}
	for {
		err := datastore.RunInTransaction(context, func(context appengine.Context) error {
			id = fmt.Sprintf("%d", rand.Int())
			key := datastore.NewKey(context, "Session", id, 0, nil)
			err := datastore.Get(context, key, &s)
			if err != datastore.ErrNoSuchEntity {
				return SessionAlreadyExists
			}
			_, err = datastore.Put(context, key, &s)
			if err != nil {
				context.Warningf("Put new empty sess. SID=%s ERR= %s", id, err.Error())
			}
			return nil
		}, nil)
		if err == SessionAlreadyExists {
			continue
		}
		if err != nil {
			context.Warningf("newSessionRecordID transaction ERR= %s", err.Error())
		}
		break
	}
	return
}

// Associate teamID with a session
func (session *Session) loginSession(context appengine.Context, teamID string) {
	s := SessionRecord{
		Created: time.Now(),
		TeamID:  teamID,
	}
	key := datastore.NewKey(context, "Session", session.ID, 0, nil)
	_, err := datastore.Put(context, key, &s)
	if err != nil {
		context.Warningf("login didn't save? SID=%s T=%s ERR=%s", session.ID, teamID, err.Error())
	}
	err = memcache.Set(context, &memcache.Item{
		Key:   "Session/" + session.ID,
		Value: []byte(teamID),
	})
	if err != nil {
		context.Warningf("login no cache SID=%s TID=%s ERR= %s", session.ID, teamID, err.Error())
	}
}

// To guard against XSRF, we use "action tokens" for some actions.
// To use these, on the form that prompts user to do something, include
// this token as a hidden value a la
//   m.token := session.actionToken("something awesome")
//   <input type="hidden" name="token" value="{{.token}}">
// Then the script that handles the form checks the token.  If it doesn't
// match, then don't carry out the tricky action.
//   if session.actionToken("something awesome") != r.FormValue("token") {
//     return // refuse to do something awesome
//   }
func (session *Session) actionToken(verb string) (token string) {
	h := adler32.New() // why adler32? it's first in the docs. (ie alpha order)
	h.Write([]byte(session.ID))
	h.Write([]byte(verb))
	return fmt.Sprintf("%d", h.Sum32())
}

func CleanupSessions(context appengine.Context) {
	ago := time.Second * -2 * SESSION_LIFETIME_S
	q := datastore.NewQuery("Session").Order("-Created").Filter("Created <", time.Now().Add(ago)).KeysOnly()
	keys, _ := q.GetAll(context, nil)
	for _, key := range keys {
		err := datastore.Delete(context, key)
		if err != nil {
			context.Warningf("CleanupSessions DELwhoops ERR=%s", err.Error())
		}
	}
}
