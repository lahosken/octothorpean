# Some handy endpoints

There's a session cookie "sid", so keep it and use it.

## /login.json?team=Amazing Team Name&password=foo42bar

Assuming there's a team with that name/password,
log in the current session w/that team.

Returned JSON on a successful login looks like

    { "message": "OK.", 
      "similars": [], 
      "success": true, 
      "team": "Amazing Team Name" }

Some not-so-successful results might be

    { "message": "No such team!",
      "similars": [ "Awesome Team Name", "Amazing Meat Name" ],
      "success": false,
      "team": "" }

or

    { "message": "Password did not match!",
      "similars": [],
      "success": false,
      "team": "Awesome Team Name" }

## /a.json?act=ui

Fetch some info useful for interacting with this puzzle.

Hang onto guesstoken and hinttoken. If the team wants to
enter a guess or ask for a hint, you'll need one of those.

guts is the HTML of the puzzle itself.

    { "act": "ui",
      "guesstoken": "5559327791",
      "guts": "blah blah peek at the \u003ca href=\"soln.html\"\u003e",
      "hints": null,
      "hinttoken": "5551748155",
      "solvedP": false,
      "title": "User Interface" }

## /guess?act=ui&guess=foo bar&token=5559327791

Guess at the answer to a puzzle. The token is that guesstoken from earlier

A response has feedback. A totally-correct response has feedback and
perhaps nextacts, a list of other act IDs unlocked.

    {
       "feedback": "You solved it! Solution was FOOBAR. Unlocked:  \u003ca href=\"/a/ock/\"\u003eock\u003c/a\u003e",
       "nextacts": [ "ock" ]
    }

A not-so successful response has feedback. It's probably worth showing
that feedback: if it's close to an answer, it'll prompt to check for typos.
Some "partial" answers give nudges, etc.

    {
      "feedback": "Morse is important, but something else is also important."
    }

## /hint?act=ui&num=1&token=5551748155

Ask for a hint. The token is that hinttoken from earlier.
The num is which hint we want: E.g., if team currently is looking at
zero hints and wants the first hint, this asks for hint 1.
(If the team has two devices, it's handy that if two folks ask for
a hint at the same time, the game doesn't jump ahead two hints.)

    {
      "hints": [ "You're holding it wrong." ]
    }

