package octo

import (
	"appengine"
	"appengine/datastore"
	"net/http"
	"strings"
)

/**
 * Misc endpoints useful for TernGame '14 and maybe the future
 */

/* To enable these features, tweak these values: */
var WOMBAT_ENABLE = false
var WOMBAT_PASSWORD = "wombat"

/**
 answer_file (I was thinking of moving the data for the location of the
next puzzle into start_codes.json and specifying sequence there):
{
 "version" : 1,
 "answer_list" : [
     { "answer" : "wombat",
     "response" : "Great guess!  But not a correct one."
      },
     {
      "answer" : "wombatwombat",
      "response" : "Now you're really cooking... keep going..."
      } ,
    {
   "answer" : "wombatwombatwombat",
   "response" :"It's like you read my mind! Head to the ice cream shop
on 501 Octavia Street, hand the woman your \
    monopoly money (that's $500 mono-bucks, yo!) and see what you get
in return.",
    "correct" : true
    }
    ]
}
*/

func wombatact(w http.ResponseWriter, r *http.Request) {
	if WOMBAT_ENABLE != true {
		spewjsonp(w, r, MapSI{"wombat": "disabled"})
		return
	}
	if WOMBAT_PASSWORD != r.FormValue("wombatpassword") {
		spewjsonp(w, r, MapSI{"wombat": "bad wombatpassword"})
		return
	}
	context := appengine.NewContext(r)
	actID := r.FormValue("act")
	if actID == "" {
		spewjsonp(w, r, MapSI{"wombat": "need act=something"})
		return
	}
	key := datastore.NewKey(context, "Activity", actID, 0, nil)
	act := ActivityRecord{}
	err := datastore.Get(context, key, &act)
	if err == datastore.ErrNoSuchEntity {
		spewjsonp(w, r, MapSI{"wombat": "no such act"})
		return
	}
	if err != nil {
		spewjsonp(w, r, MapSI{"wombat": "err: " + err.Error()})
		return
	}
	var answer_list = []MapSI{}
	for ix, val := range act.Solutions {
		answer_list = append(answer_list, MapSI{
			"answer":    val,
			"correct":   true,
			"canonical": (ix == 0), // first answer in puz.txt is canonical
		})
	}
	for _, val := range act.Partials {
		split := strings.SplitN(val, " ", 2)
		if len(split) == 2 {
			answer_list = append(answer_list, MapSI{
				"answer":   split[0],
				"response": split[1],
				"correct":  false,
			})
		} else {
			answer_list = append(answer_list, MapSI{
				"answer":  split[0],
				"correct": false,
			})
		}
	}
	retval := MapSI{
		"raw":         act,
		"answer_list": answer_list,
	}
	for _, val := range act.Extras {
		split := strings.SplitN(val, " ", 2)
		retval[split[0]] = split[1]
	}
	spewjsonp(w, r, retval)
}

/**
start_codes.json:
{
  "version" : 1,
  "start_codes" : [
     {
       "id" : "puzzle1",
       "name" : "orienteering",
       "answer_file" : "puzzle1_answers.json"

     },
     {
       "id" : "wombat",
       "name" : "God save the wombats",
       "answer_file" : "wombat_answers.json"
     }
   ]
}
*/
func wombatarc(w http.ResponseWriter, r *http.Request) {
	if WOMBAT_ENABLE != true {
		spewjsonp(w, r, MapSI{"wombat": "disabled"})
		return
	}
	if WOMBAT_PASSWORD != r.FormValue("wombatpassword") {
		spewjsonp(w, r, MapSI{"wombat": "bad wombatpassword"})
		return
	}
	context := appengine.NewContext(r)
	arcID := r.FormValue("arc")
	arc := fetcharc(context, arcID)
	retval := []MapSI{}
	for _, act := range arc.Act {
		retval = append(retval, MapSI{
			"id": act,
		})
	}
	spewjsonp(w, r, retval)
}
