// file: retrieve.go
package main

import (
	"fmt"
	"gopkg.in/xmlpath.v2"
	"log"
	"net/http"
	"strings"
)

const (
	nothing = iota
	inTable
	inTR
	inTD
)

type RenamePattern struct {
	To   string
	From []string
}

var gotit bool = false

type match struct {
	from string
	to   string
}

var (
	mWC = []match{}
	mDR = []match{}

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

	GenreSubst = map[string]string{
		"cпецслужбы":                   "military_special",
		"автомобили и пдд":             "auto_regulations",
		"альтернативная история":       "sf_history",
		"альтернативная медицина":      "sci_medicine_alternative",
		"аналитическая химия":          "sci_anachem",
		"анекдоты":                     "humor_anecdote",
		"антисоветская литература":     "dissident",
		"античная литература":          "antique_ant",
		"аппаратное обеспечение":       "comp_hard",
		"архитектура":                  "architecture_book",
		"астрология":                   "astrology",
		"астрономия и космос":          "sci_cosmos",
		"афоризмы":                     "aphorisms",
		"базы данных":                  "comp_db",
		"банковское дело":              "banking",
		"басни":                        "fable",
		"биографии и мемуары":          "nonf_biography",
		"биология":                     "sci_biology",
		"биофизика":                    "sci_biophyƒs",
		"биохимия":                     "sci_biochem",
		"боевая фантастика":            "sf_action",
		"боевик":                       "det_action",
		"боевые искусства":             "military_arts",
		"ботаника":                     "sci_botany",
		"буддизм":                      "religion_budda",
		"бухучет и аудит":              "accounting",
		"былины":                       "epic",
		"в стихах":                     "in_verse",
		"верлибры":                     "vers_libre",
		"вестерн":                      "adv_western",
		"ветеринария":                  "sci_veterinary",
		"визуальная поэзия":            "visual_poetry",
		"внешняя торговля":             "global_economy",
		"водевиль":                     "vaudeville",
		"военная документалистика":     "nonf_military",
		"военная история":              "military_history",
		"военная техника и вооружение": "military_weapon",
		"военное дело: прочее":         "military",
		"газеты и журналы":             "periodic",
		"геология и география":         "sci_geo",
		"героическая фантастика":       "sf_heroic",
		"городское фэнтези":            "sf_fantasy_city",
		"государство и право":          "sci_state",
		"готический роман":             "gothic_novel",
		"дамский детективный роман":    "det_cozy",
		"деловая литература":           "sci_business",
		"делопроизводство":             "paper_work",
		"детективная фантастика":       "sf_detective",
		"детективы: прочее":            "detective",
		"детская литература: прочее":   "children",
		"детская проза":                "child_prose",
		"детская психология":           "psy_childs",
		"детская фантастика":           "child_sf",
		"детские остросюжетные":        "child_det",
		"детские приключения":          "child_adv",
		"детские стихи":                "child_verse",
		"детский фольклор":             "child_folklore",
		"документальная литература":    "nonfiction",
		"домашние животные":            "home_pets",
		"домоводство":                  "home",
		"драма":                        "drama",
		"драматургия: прочее":          "dramaturgy",
		"древневосточная литература":   "antique_east",
		"древнеевропейская литература": "antique_european",
		"древнерусская литература":     "antique_russian",
		"загадки":                      "riddles",
		"здоровье":                     "home_health",
		"зоология":                     "sci_zoo",
		"изобразительное искусство, фотография": "visual_arts",
		"индуизм":                         "religion_hinduism",
		"иностранные языки":               "foreign_language",
		"интернет":                        "comp_www",
		"ироническая фантастика":          "sf_irony",
		"иронический детектив":            "det_irony",
		"ироническое фэнтези":             "sf_fantasy_irony",
		"искусство и дизайн":              "design",
		"ислам":                           "religion_islam",
		"историческая проза":              "prose_history",
		"исторические любовные романы":    "love_history",
		"исторические приключения":        "adv_history",
		"исторический детектив":           "det_history",
		"историческое фэнтези":            "historical_fantasy",
		"история":                         "sci_history",
		"иудаизм":                         "religion_judaism",
		"католицизм":                      "religion_catholicism",
		"киберпанк":                       "sf_cyberpunk",
		"кино":                            "cine",
		"киносценарии":                    "screenplays",
		"классическая проза":              "prose_classic",
		"классический детектив":           "det_classic",
		"книга-игра":                      "prose_game",
		"коллекционирование":              "home_collecting",
		"комедия":                         "comedy",
		"контркультура":                   "prose_counter",
		"короткие любовные романы":        "love_short",
		"корпоративная культура":          "org_behavior",
		"космическая фантастика":          "sf_space",
		"космоопера":                      "sf_space_opera",
		"криминальный детектив":           "det_crime",
		"критика":                         "nonf_criticism",
		"крутой детектив":                 "det_hard",
		"кулинария":                       "home_cooking",
		"культурология":                   "sci_culture",
		"лирика":                          "lyrics",
		"литературоведение":               "sci_philology",
		"личные финансы":                  "personal_finance",
		"любовная фантастика":             "love_sf",
		"любовные детективы":              "love_detective",
		"магический реализм":              "prose_magic",
		"малый бизнес":                    "small_business",
		"маньяки":                         "det_maniac",
		"маркетинг, pr, реклама":          "marketing",
		"математика":                      "sci_math",
		"медицина":                        "sci_medicine",
		"медицинский триллер":             "thriller_medical",
		"металлургия":                     "sci_metal",
		"мистерия":                        "mystery",
		"мистика":                         "sf_mystic",
		"мифы. легенды. эпос":             "antique_myths",
		"морские приключения":             "adv_maritime",
		"музыка":                          "music",
		"народные песни":                  "folk_songs",
		"народные сказки":                 "folk_tale",
		"научная литература: прочее":      "science",
		"научная фантастика":              "sf",
		"научпоп":                         "sci_popular",
		"недвижимость":                    "real_estate",
		"недописанное":                    "unfinished",
		"ненаучная фантастика":            "nsf",
		"неотсортированное":               "other",
		"новелла":                         "story",
		"о бизнесе популярно":             "popular_business",
		"о войне":                         "prose_military",
		"о любви":                         "love",
		"образовательная литература":      "child_education",
		"обществознание":                  "sci_social_studies",
		"околокомпьютерная литература":    "computers",
		"органическая химия":              "sci_orgchem",
		"ос и сети":                       "comp_osnet",
		"отраслевые издания":              "industries",
		"палиндромы":                      "palindromes",
		"партитуры":                       "notes",
		"педагогика":                      "sci_pedagogy",
		"песенная поэзия":                 "song_poetry",
		"повесть":                         "great_story",
		"подростковая литература":         "ya",
		"поиск работы, карьера":           "job_hunting",
		"политика":                        "sci_politics",
		"политический детектив":           "det_political",
		"полицейский детектив":            "det_police",
		"попаданцы":                       "popadanec",
		"порно":                           "love_hard",
		"пословицы, поговорки":            "proverbs",
		"постапокалипсис":                 "sf_postapocalyptic",
		"поэзия: прочее":                  "poetry",
		"православие":                     "religion_orthodoxy",
		"приключения про индейцев":        "adv_indian",
		"приключения: прочее":             "adventure",
		"природа и животные":              "adv_animal",
		"программирование":                "comp_programming",
		"программы":                       "comp_soft",
		"проза":                           "prose",
		"протестантизм":                   "religion_protestantism",
		"психология":                      "sci_psychology",
		"психотерапия и консультирование": "psy_theraphy",
		"публицистика":                    "nonf_publicism",
		"путеводители":                    "geo_guides",
		"путешествия и география":         "adv_geo",
		"радиоэлектроника":                "sci_radio",
		"развлечения":                     "home_entertain",
		"рассказ":                         "short_story",
		"религиоведение":                  "sci_religion",
		"религиозная литература: прочее":  "religion",
		"религия":                         "religion_rel",
		"рефераты":                        "sci_abstract",
		"роман":                           "roman",
		"руководства":                     "ref_guide",
		"русская классическая проза":      "prose_rus_classic",
		"сад и огород":                    "home_garden",
		"самосовершенствование":           "religion_self",
		"сатира":                          "humor_satire",
		"сделай сам":                      "home_diy",
		"секс и семейная психология":      "psy_sex_and_family",
		"семейный роман/семейная сага":    "sagas",
		"сентиментальная проза":           "prose_sentimental",
		"сказка":                          "child_tale",
		"сказочная фантастика":            "fairy_fantasy",
		"словари":                         "ref_dict",
		"советская классическая проза":    "prose_su_classics",
		"современная проза":               "prose_contemporary",
		"социальная фантастика":           "sf_social",
		"спорт":                           "home_sport",
		"справочная литература":           "reference",
		"справочники":                     "ref_ref",
		"старинная литература: прочее":    "antique",
		"стимпанк":                        "sf_stimpank",
		"строительство и сопромат":        "sci_build",
		"сценарии":                        "scenarios",
		"театр":                           "theatre",
		"технические науки":               "sci_tech",
		"техно триллер":                   "thriller_techno",
		"технофэнтези":                    "sf_technofantasy",
		"торговля":                        "trade",
		"трагедия":                        "tragedy",
		"транспорт и авиация":             "sci_transport",
		"триллер":                         "thriller",
		"ужасы":                           "sf_horror",
		"управление, подбор персонала": "management",
		"учебники":                     "sci_textbook",
		"фантастика: прочее":           "sf_etc",
		"фанфик":                       "fanfiction",
		"феерия":                       "extravaganza",
		"физика":                       "sci_phys",
		"физическая химия":             "sci_physchem",
		"философия":                    "sci_philosophy",
		"фольклор: прочее":             "folklore",
		"фэнтези":                      "sf_fantasy",
		"химия":                        "sci_chem",
		"хиромантия":                   "palmistry",
		"хобби и ремесла":              "home_crafts",
		"христианство":                 "religion_christianity",
		"ценные бумаги, инвестиции":    "stock",
		"цифровая обработка сигналов":  "comp_dsp",
		"частушки, прибаутки, потешки": "limerick",
		"шпаргалки":                    "sci_crib",
		"шпионский детектив":           "det_espionage",
		"эзотерика":                    "religion_esoterics",
		"экология":                     "sci_ecology",
		"экономика":                    "economics",
		"экспериментальная поэзия":     "experimental_poetry",
		"энциклопедии":                 "ref_encyc",
		"эпистолярная проза":           "epistolary_fiction",
		"эпическая поэзия":             "epic_poetry",
		"эпическая фантастика":         "sf_epic",
		"эпопея":                       "prose_epic",
		"эротика":                      "love_erotica",
		"эротика, секс":                "home_sex",
		"эссе, очерк, этюд, набросок":  "essay",
		"юмор: прочее":                 "humor",
		"юмористическая проза":         "humor_prose",
		"юмористическая фантастика":    "sf_humor",
		"юмористические стихи":         "humor_verse",
		"юмористическое фэнтези":       "humor_fantasy",
		"юридический триллер":          "thriller_legal",
		"юриспруденция":                "sci_juris",
		"языкознание":                  "sci_linguistic",
		"язычество":                    "religion_paganism",
		"современные любовные романы":  "love_contemporary",
	}
)

func (book *Book) SearchLitMir() {
	// 	walk("http://www.litmir.co/bd/?b=253328", b)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("http transport error is:", err)
	}
	root, err := xmlpath.ParseHTML(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	title := xmlpath.MustCompile("//h1[@class='lt35']")
	series := xmlpath.MustCompile("//b[contains(.,'Серии:')]/following-sibling::a")
	// author := xmlpath.MustCompile("//b[contains(.,'Автор:')]/following-sibling::a")
	genre := xmlpath.MustCompile("//b[contains(.,'Жанр:')]/following-sibling::a")

	if i := title.Iter(root); i != nil {
		for i.Next() {
			n := i.Node()
			book.Title = fmt.Sprintf("%s", n)
		}
	}

	if i := series.Iter(root); i != nil {
		for i.Next() {
			n := i.Node()
			book.Sequences = append(book.Sequences, Sequence{fmt.Sprintf("%s", n), 0})
		}
	}
	if i := genre.Iter(root); i != nil {
		for i.Next() {
			gen := strings.ToLower(fmt.Sprintf("%s", i.Node()))
			book.Genres = append(book.Genres, GenreSubst[gen])
		}
	}
	// if i := author.Iter(root); i != nil {
	// 	for i.Next() {
	// 		n := i.Node()
	// 		book.Authors = append(book.Authors, fmt.Sprintf("%s", n))
	// 	}
	// }
}

// func main() {
//
// 	fmt.Printf("Authors: '%v'\n", b.Authors)
// 	fmt.Printf("Title: '%s'\n", b.Title)
// 	fmt.Printf("Series: '%s'\n", b.Sequences)
// 	fmt.Printf("Genres: '%s'\n", b.Genres)
//
func SearchLibRusEc(book *Book) *book {
	// doc, err := goquery.NewDocument("http://lib.rus.ec/b/553014")
	// if err != nil {
	// log.Fatal(err)
	// }

	//book[author/@id='CMS']/title                   =>  "Being a Dog Is a Full-Time Job",
	// /library/book/preceding::comment()               =>  " Great book. "
	//*[contains(born,'1922')]/name                  =>  "Charles M Schulz"
	//*[@id='PP' or @id='Snoopy']/born

	// doc.Find("._ga1_on_").Each(func(i int, s *goquery.Selection) {
	// fmt.Println("YO")
	// fmt.Println(s.Find("h9").Text().Each())
	// title := s.Find("i").Text()
	// fmt.Printf("Review %d: %s - %s\n", i, band, title)
	// })

	// fmt.Println(path.Iter(root).Next()) // {
	// fmt.Println("Found:", value)
	// }

	// // fmt.Println("read error is:", err)

	// // fmt.Println(string(body))

	// doc, err := html.Parse(resp.Body)
	// if err != nil {
	// 	fmt.Println("http transport error 2 is:", err)
	// 	// ...
	// }
	// walk(doc)
	// var f func(*html.Node)

	// f = func(n *html.Node) {
	// 	if n.Type == html.ElementNode && n.Data == "a" {
	// 		for _, a := range n.Attr {
	// 			if a.Key == "href" {
	// 				fmt.Println(a.Val)
	// 				break
	// 			}
	// 		}
	// 	}
	// 	for c := n.FirstChild; c != nil; c = c.NextSibling {
	// 		f(c)
	// 	}
	// }
	// f(doc)

	// z := html.NewTokenizer(resp.Body)
	// depth := 0
	// for {
	// 	tt := z.Next()
	// 	switch tt {
	// 	case html.ErrorToken:
	// 		fmt.Println(z.Err())
	// 		return
	// 	case html.TextToken:
	// 		if depth > 0 {
	// 			// emitBytes should copy the []byte it receives,
	// 			// if it doesn't process it immediately.
	// 			// emitBytes(z.Text())
	// 			fmt.Println("deeper", depth)
	// 		}
	// 	case html.StartTagToken, html.EndTagToken:
	// 		tn, _ := z.TagName()
	// 		if len(tn) == 1 && tn[0] == 'a' {
	// 			if tt == html.StartTagToken {
	// 				depth++
	// 			} else {
	// 				depth--
	// 			}
	// 		}
	// 	}
	// }
}

// }
