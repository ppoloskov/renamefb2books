package main

import (
	"encoding/json"
	"fmt"
)

var (
	Patterns = []RenamePattern{
		RenamePattern{To: "business", From: []string{"accounting", "banking", "economics", "global_economy", "industries", "job_hunting", "management",
			"marketing", "org_behavior", "paper_work", "personal_finance", "popular_business", "real_estate",
			"sci_business", "small_business", "stock", "trade"}},
		RenamePattern{To: "classic", From: []string{"*_classic", "antique*", "folk*", "prose_classic"}},
		RenamePattern{To: "fiction", From: []string{"adv*", "aphorisms", "*_contemporary", "comedy", "det*", "dissident", "drama", "epic", "epic_poetry", "epistolary_fiction",
			"essay", "experimental_poetry", "extravaganza", "fable", "fanfiction", "gothic_novel", "great_story", "in_verse", "limerick", "love*",
			"lyrics", "mystery", "palindromes", "prose*", "roman", "sagas", "scenarios", "screenplays", "short_story", "song_poetry", "story",
			"thriller", "tragedy", "vaudeville", "vers_libre", "ya"}},
		RenamePattern{To: "fs&f", From: []string{"child*", "fairy_fantasy", "historical_fantasy", "nsf", "popadanec", "sf*", "humor*"}},
		RenamePattern{To: "nonfiction", From: []string{"comp*", "home*", "military*", "music", "nonf_criticism", "nonf_military", "nonf_publicism",
			"nonfiction", "psy*", "religion*", "sci*"}},
		RenamePattern{To: "Nonfiction/Biography", From: []string{"nonf_biography"}},
		RenamePattern{To: "Nonfiction/History", From: []string{"sci_history"}},
		RenamePattern{To: "other", From: []string{"astrology", "auto_regulations", "other", "ref*"}},
	}
)

type User struct {
	Name string
}

type RenamePattern struct {
	From []string
	To   string
}

func main() {

	var Match = make(map[string]string)

	for _, p := range Patterns {
		for _, mt := range p.From {
			Match[mt] = p.To

		}
	}

	b, err := json.MarshalIndent(Patterns, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(b))
}# 
