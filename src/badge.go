package octo

import (
	"appengine"
	"appengine/datastore"
	"html/template"
//	"log"
	"math"
	"net/http"
	"net/url"
)

type BadgeBling struct {
	Pretty string
	Description string
}
var badgeBling = map[string]BadgeBling {
	"act": {
		"Active",
	    `You earn this badge by solving many many puzzles.`,
	},
	"5bit": {
		"5 Bit Binary",
	    `You earn this badge by using 5-bit binary numbers. This is
		 a popular code: 5 bits is all you need to encode the numbers
		1-26, and thus the letters of the alphabet.`,
	},
	"a1": {
		"1=A, 2=B, 3=C, ...",
	    `You earn this badge by decoding numbers to get letters using
		the popular code 1=A, 2=B, 3=C, ..., 26=Z.`,
	},
	"anagram": {
		"Anagram",
	    `You earn this badge by unscrambling anagrams, a great art.`,
	},
	"braille": {
		"Braille",
	    `You earn this badge by decoding messages in Braille code.`,
	},
	"calendar": {
		"Calendar",
	    `You earn this badge by solving calendric puzzles. The
		 <a href="http://puzzlehuntcalendar.org/" target="_blank">Puzzlehunt
         Calendar</a> shows you upcoming puzzlehunt events, calendric and
         otherwise (mostly otherwise).`,
	},
	"cipher": {
		"Cipher",
	    `You earn this badge by deciphering messages. Substitution ciphers
         are tricky. It's usually easier to deduce the key of
         longer messages than of short ones; but it's a lot of work to
         decipher a long message even if you know the key.`,
	},
	"code": {
		"Code",
	    `You earn this badge by decoding encoded messages.`,
	},
	"conspiracy": {
		"#Conspiracy",
	    `You earn this badge by unraveling the secret of the
         Octothorpean Order's Conspiracy.`,
	},
	"crossword": {
		"Crossword",
	    `You earn this badge through feats of cruciverbalism, i.e., 
        solving crossword puzzles. Puzzlehunt crossword puzzles tend
        to be trickier than normal. Even after you fill in the grid,
        you probably still need to extract some answer-phrase.`,
	},
	"dash": {
		"DASH",
	    `You earn this badge for solving a variety of puzzles.
         <a href="http://playdash.org" target="_blank">DASH</a>
         (Different Area, Same Hunt) is an annual puzzlehunt event.
		 It takes place in several cities, perhaps including
         your hometown.`,
	},
	"electronic": {
		"Electronic",
	    `You earn this badge by solving puzzles favored by folks
        who enjoy soldering wires to things.`,
	},
	"guardian": {
		"#Guardian",
	    `You earn this badge by defeating one of the Guardians
        of the <a href="/b/conspiracy">Conspiracy</a>. Each
        Guardian lurks behind several puzzles.`,
	},
	"location": {
		"Location, Location, Location",
	    `You earn this badge by going places.`,
	},
	"logic": {
		"Logic",
	    `You earn this badge by solving logic puzzles.`,
	},
	"morse": {
		"Morse",
	    `You earn this badge by decoding Morse code messages.`,
	},
	"music": {
		"Music",
	    `You earn this badge by solving puzzles involving music.
		It's like easy listening, but difficult.`,
	},
	"nikoli": {
		"Nikoli",
	    `You earn this badge by solving puzzles in the Nikoli style.
		<a href="http://www.nikoli.com/">Nikoli</a>
        makes logic puzzles. They are a Japanese company, but
        their puzzles do not require you to know Japanese&mdash;their
        puzzles tend not to use words at all.`,
	},
	"numeric": {
		"Numeric",
	    `You earn this badge by solving puzzles that use numbers.
		 Letters are for chumps!`,
	},
/* Never heard back from him
	"panda": {
		"P&A",
	    `You earn this badge by solving puzzles from
        <a href="http://www.pandamagazine.com/">Panda Magazine</a>,
        an online puzzlehunt that comes along every two months.`,
	},
*/
	"pint": {
		"Puzzled Pint",
	    `You earn this badge by solving puzzles designed by
        <a href="http://www.puzzledpint.com/">Puzzled Pint</a>, folks
        who get together monthly in Portland OR and Seattle WA to
        enjoy puzzles and refreshing beverages.`,
	},
	"playtest": {
		"Playtesting",
	    `You think you had a hard time solving these puzzles?
         Playtesters had to solve these puzzles back when the
         puzzles were <em>broken</em>.`,
	},
	"poem": {
		"Poem",
	    `Whether your astrological sign is Libra or Sag-<br>
			gitarius; no matter what stars your homes,<br>
		you can earn this excellent badge,<br>
		 by enduring some awful poems.`,
	},
	"popculture": {
		"Pop Culture",
	    `You earn this badge by solving puzzles involving popular culture.
		At last, you've found a purpose for your movie trivia knowledge.`,
	},
//	"puzzazz": {
//		"Puzzazz",
//	    `You earn this badge by solving puzzles in the style of
//        <a href="http://www.puzzazz.com/puzzles" target="_blank">Puzzazz's
//        puzzle of the day</a>.
//			<a href="http://www.Puzzazz.com/" target="_blank">Puzzazz</a> is a puzzle and
//        technology company. They run internet puzzlehunts, make Kindle
//        Sudoku apps, and do stranger things.`,
//	},
	"ravenchase": {
		"Ravenchase",
	    `You earn this badge by solving puzzles in the style of
        <a href="http://www.ravenchase.com/" target="_blank">Ravenchase
        Adventures</a>.
        They run puzzlehunt events featuring treasure maps, invisible ink,
        classic codes, and historic sites.`,
	},
	"rdu": {
        "Research Triangle",
        `You earn this badge by solving puzzles that use parts of 
        the Raleigh-Durham area, land of the 
        <a href="http://duke-dagger.blogspot.com/p/puzzlehunt-registration.html">DAGGER puzzlehunt</a>.`,
	},
	"semaphore": {
		"Semaphore",
	    `You earn this badge by decoding messages using semaphore flags.`,
	},
	"sfo": {
        "San Francisco Bay Area",
        `You earn this badge by solving puzzles that use parts of 
        San Francisco, land of the 
        <a href="http://bayareanightgame.org/">Bay Area Night Game</a> and the
        <a href="http://www.2tonegame.org/">2-Tone Game</a>.`,
	},
	"shinteki": {
		"Shinteki",
	    `You earn this badge by solving puzzles in the Shinteki style.
        <a href="http://www.shinteki.com">Shinteki</a> runs live puzzlehunt
        events. They run corporate team-building exercises that your
        coworkers can understand; they run trickier events for by and about
        puzzlehunters. They welcome puzzling beginners and experts.`,
	},
	"word": {
		"Word",
	    `You earn this badge by solving word puzzles.`,
	},
}

func newBadges(actTags []string, points map[string]int, already map[string] int) (newbadges map[string]int) {
	newbadges = map[string]int{}
	var actTagsSet = map[string]bool{}
	for _, tag := range actTags { actTagsSet[tag] = true }

	// Some high-priority things to try first
	if (points["guardian"] > already["guardian"]) { newbadges["guardian"] = points["guardian"] }
	for _, badge := range []string{ "conspiracy", "shinteki" } {
		if (points[badge] > already[badge]) { newbadges[badge] = points[badge] }
	}
	if len(newbadges) > 0 { return }
	// Some badges for "organizations" (shinteki's above) and locations:
	var orgRatios = map[string]int {
		"playtest": 1, // not really an org, but this is a handy place
		"calendar": 1,
		"nikoli": 2,
		"npl": 1,
//		"panda": 2, // never heard back from them
		"pint": 1,
//		"puzzazz": 2, // they don't wanna badge
		"ravenchase": 1,
		"rdu": 2,
		"sfo": 2,
	}
	for badge, div := range orgRatios {
		if actTagsSet[badge] && (points[badge]/div > already[badge]) { 
			newbadges[badge] = points[badge]/div 
		}
	}
	if len(newbadges) > 0 { return }
	// There's not really a DASH "style" of puzzle. So we award this badge
	// as the team solves a variety of puzzles.
    dashmult := (points["code"] - 1) * points["word"] *  (points["logic"] + 1)
	dashscore := 0
	if (dashmult) > 0 {
		dashscore = int((math.Cbrt(float64(dashmult)) - 1.5) / 2.5)
		if dashscore > already["dash"] { newbadges["dash"] = dashscore }
	}
	if len(newbadges) > 0 { return }
	// The not-uncommon types
	var ratios = map[string]int {
		"5bit": 2,
		"a1": 3,
		"anagram": 2,
		"braille": 2,
		"cipher": 2,
		"crossword": 2,
		"morse": 2,
		"semaphore": 2,
	}
	for badge, div := range ratios {
		if actTagsSet[badge] && (points[badge]/div > already[badge]) { 
			newbadges[badge] = points[badge]/div 
		}
	}
	if len(newbadges) > 0 { return }
	// some silly types
	var sillyRatios = map[string]int {
		"electronic": 2,
		"location": 3,
		"numeric": 3,
		"poem": 3,
	}
	for badge, div := range sillyRatios {
		if actTagsSet[badge] && (points[badge]/div > already[badge]) { 
			newbadges[badge] = points[badge]/div 
		}
	}
	if len(newbadges) > 0 { return }
	// some more generic types
	var genericRatios = map[string]int {
		"code": 5,
		"word": 5,
		"logic": 5,
		"music": 2,
		"popculture": 2,
	}
	for badge, div := range genericRatios {
		if actTagsSet[badge] && (points[badge]/div > already[badge]) { 
			newbadges[badge] = points[badge]/div 
		}
	}
	if len(newbadges) > 0 { return }
	// most generic: straight-up puzzle count
	if points["act"]/10 > already["act"] { newbadges["act"] = points["act"]/10 }
	return
}

// Get gossip for one badge. Handy for displaying on its profile page.
func getBadgeGossip(context appengine.Context, bid string) (out []tidbit) {
	alreadySet := make(map[string]bool) // say something once, why say it again?
	q := datastore.NewQuery("TLog").Order("-Created").Filter("Notes=", bid).Filter("Verb=", "badge").Limit(100)
	for iter := q.Run(context); ; { // TODO GetAll didn't work in r58
		var tlr TLogRecord
		_, err := iter.Next(&tlr)
		if err == datastore.Done {
			break
		}
		if err != nil {
			context.Warningf("BadgeGossip iter ERR %s", err.Error())
			break
		}
		if alreadySet[tlr.TeamID] {
			continue
		}
		alreadySet[tlr.TeamID] = true
		out = append(out, tidbit{
			T: tlr.Created.Unix() * 1000,
			M: tlr.TeamID,
		})
		if len(out) > 50 {
			break
		}
	}
	return
}

func badgeprofile(w http.ResponseWriter, r *http.Request) {
	badgeID, _ := url.QueryUnescape(r.URL.Path[len("/b/"):])
	if badgeBling[badgeID].Pretty == "" { badgelist(w, r); return }
	_, tid := GetAndOrUpdateSession(w, r)
	context := appengine.NewContext(r)
	g := getBadgeGossip(context, badgeID)
	template.Must(template.New("").Parse(tBadgeProfile)).Execute(w, MapSI{
		"PageTitle":   "Badge: " + badgeBling[badgeID].Pretty,
		"BadgeID":     badgeID,
		"Pretty":      badgeBling[badgeID].Pretty,
		"Description": template.HTML(badgeBling[badgeID].Description),
		"TID":         tid,
		"Gossip":      g,
	})
}

func badgelist(w http.ResponseWriter, r *http.Request) {
	_, tid := GetAndOrUpdateSession(w, r)
	template.Must(template.New("").Parse(tBadgeList)).Execute(w, MapSI{
		"PageTitle":   "Merit Badges",
		"TID":         tid,
		"Badges":      badgeBling,
	})
}