package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	// "github.com/metakeule/places"
)

// TODO: Add output path, sort authors in books, cleanup series, create series folders

var (
	indir           = flag.String("in", "", "Path to unsorted books")
	outdir          = flag.String("out", "./out", "Where to renamed books")
	authcompilation = flag.Int("comp", 2, "Number of authors is book file to set author to compilaton")
	rename          = flag.Bool("R", false, "Rename files")
	configfile      = flag.String("c", "config.yaml", "Path to config file")
	cpuprofile      = flag.String("cpuprofile", "", "write cpu profile to file")
	printconfig     = flag.Bool("C", false, "Print parsed config variables (debug mostly)")
	workersno       = 5
)

func findfiles(searchpath string) chan string {
	count := 0
	out := make(chan string)
	go func() {
		filepath.Walk(searchpath, func(path string, f os.FileInfo, _ error) error {
			if strings.HasSuffix(f.Name(), ".fb2") || strings.HasSuffix(f.Name(), ".zip") {
				out <- path
				count = count + 1
				fmt.Printf("Files: %d, queue len: %d [%s]\n", count, len(out), path)
			}
			return nil
		})
		close(out)
	}()
	return out
}

func worker(ID int, files chan string, out chan *Book) {
	fmt.Printf("Worker %d launched\n", ID)
	count := 0
	for path := range files {
		out <- parsefb2(path)
		count = count + 1
		fmt.Printf("Worker %d: %d\n", ID, count)
	}
	fmt.Printf("Worker %d is finished his job\n", ID)
	out <- nil
}

func main() {
	flag.Parse()
	// Profiler support
	if *indir == "" {
		fmt.Println("You must specify path to books")
		os.Exit(-1)
	}
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	conf, err := readConfig(*configfile)
	if err != nil {
		panic(err)
	}

	if *printconfig {
		fmt.Println("CMD var indir: ", indir)
		fmt.Println("CMD var outdir: ", outdir)
		fmt.Println("CMD var authcompilan: ", authcompilation)
		fmt.Println("CMD var rename: ", rename)
		fmt.Println("CMD var configfile: ", configfile)
		fmt.Println("CMD var cpuprofile: ", cpuprofile)
		fmt.Println("CMD var printconfig: ", printconfig)
		fmt.Println("CMD var workersno: ", workersno)
		fmt.Println("CONF var Auto compilation ", conf.Authcompilation)
		fmt.Println("CONF vars GenreRenameRules, Direct rules:")
		for _, v := range conf.DirectRules {
			fmt.Println(v)
		}
		fmt.Println("CONF vars GenreRenameRules, Wildcard rules:")
		for _, v := range conf.WildRules {
			fmt.Println(v)
		}
	}
	fmt.Println("Looking for files in ", *indir)

	var wg sync.WaitGroup

	// Make channels for incoming file names and parced books
	FilesQueue := findfiles(*indir)
	BooksQueue := make(chan *Book)

	for i := 0; i < workersno; i++ {
		wg.Add(1)
		go worker(i, FilesQueue, BooksQueue)
	}

	GoodBooks := []*Book{}
	NoIDBooks := []*Book{}
	ErrorBooks := []*Book{}

	db, _ := OpenDB("")
	IDCount := 0
	NoIDConut := 0
	AuthorsCounter := make(map[int]*Person)

	// Read all book files, sort correct and incorrect and accumulate authors info for list of corrections
	go func() {
		for b := range BooksQueue {
			if b == nil {
				wg.Done()
				continue
			}
			if b.Error != nil {
				log.Println("Error processing book %s: %s", b.Path, b.Error)
				ErrorBooks = append(ErrorBooks, b)
				continue
			}

			if err := b.GetIDbyMD5(db); err != nil {
				log.Println(err)
			}
			if b.ID < 1 {
				if err := b.GetIDbyTitle(db); err != nil {
					if err == ErrNoID {
						fmt.Println("YO!")
					}
					log.Println(err)
				}
				NoIDBooks = append(NoIDBooks, b)
				fmt.Printf("No ID found for: %s\n", b.Path)
				continue
			}
			if err := b.GetTitle(db); err != nil {
				log.Println(err)
			}
			if err := b.GetAuthors(db); err != nil {
				log.Println(err)
			}
			if err := b.GetSeries(db); err != nil {
				log.Println(err)
			}

			if b.ID > 0 {
				GoodBooks = append(GoodBooks, b)
			} else {
				NoIDBooks = append(NoIDBooks, b)
				fmt.Printf("No ID found for: %s\n", b.Path)
				continue
			}
			// for _, a := range b.Authors {
			// 	authorscounter[a]++
			// }
		}
	}()
	wg.Wait()

	fmt.Printf("Good books: %d, books with ISs: %d, noISs: %d\n", len(GoodBooks), IDCount, NoIDConut)

	AuthorSequencesCounter := map[string]int{}
	PublisherSequencesCounter := map[string]int{}

	for _, b := range GoodBooks {
		fmt.Printf("%s\n", b)
		for i, a := range b.CorrectAuthors {
			if _, ok := AuthorsCounter[a.LRSId]; !ok {
				AuthorsCounter[a.LRSId] = &b.CorrectAuthors[i]
				AuthorsCounter[a.LRSId].BookCounter = 1
			} else {
				AuthorsCounter[a.LRSId].BookCounter++
			}
		}
		for _, s := range b.CorrectAuthorSequences {
			AuthorSequencesCounter[s.Name]++
		}
		for _, s := range b.CorrectPublisherSequences {
			PublisherSequencesCounter[s.Name]++
		}
	}

	for k, a := range AuthorsCounter {
		if err := AuthorsCounter[k].GetPopGenres(db); err != nil {
			log.Println(err)
		}
		fmt.Println(a)
	}

	fmt.Println("---")
	fmt.Println(AuthorSequencesCounter)
	fmt.Println("---")
	fmt.Println(PublisherSequencesCounter)
	// for k, v := range AuthorsCounter {
	// 	fmt.Printf("key[%s] value[%v]\n", k, v)
	// }

	// Create list of authors substitutions
	// AuthorsReplaceList := GenerateAuthorReplace(authorscounter)

	// type pg struct {
	// 	a Person
	// 	g string
	// }

	// authgencounter := make(map[Person]map[string]int)
	// AuthorOrigin := make(map[Person]map[string]int)
	// AuthorSequence := make(map[Person]map[string]int)
	// 		if repl, ok := AuthorsReplaceList[a]; ok {
	// 			b.Authors[i] = repl
	// 		}
	// 		AuthorsBookCounter[b.Authors[i]]++
	// 		for _, g := range b.Genres {
	// 			if g == "" {
	// 				continue
	// 			}
	// 			if _, ok := authgencounter[b.Authors[i]]; !ok {
	// 				authgencounter[b.Authors[i]] = make(map[string]int)
	// 			}
	// 			authgencounter[b.Authors[i]][g]++
	// 		}
	// 		if _, ok := AuthorOrigin[b.Authors[i]]; !ok {
	// 			AuthorOrigin[b.Authors[i]] = make(map[string]int)
	// 		}
	// 		AuthorOrigin[b.Authors[i]][b.Translated()]++
	// 		for _, s := range b.AuthorSequences {
	// 			if _, ok := AuthorSequence[b.Authors[i]]; !ok {
	// 				AuthorSequence[b.Authors[i]] = make(map[string]int)
	// 			}
	// 			AuthorSequence[b.Authors[i]][s.Name]++
	// for seq, num := range SequenceCounter {
	// 	if num > 10 {
	// 		fmt.Println("S: ", seq, num)
	// 	}
	// }
	// fmt.Printf("Found %d books and %d books with errors\n", len(GoodBooks), len(ErrorBooks))

	// // jsonExport(AuthorsReplaceList, "a.json")

	// // db, _ := CreateDB("")
	// // for _, b := range GoodBooks {
	// // 	AddBookDB(db, b)
	// // }

	// // AuthorGenre := make(map[Person]string)

	// for aut, gen := range authgencounter {
	// 	bestgen := ""
	// 	count := 0
	// 	for g, c := range gen {
	// 		if c > count {
	// 			bestgen = g
	// 			count = c
	// 		}
	// 	}
	// 	if gs.Replace(bestgen) != "" {
	// 		AuthorGenre[aut] = strings.ToLower(gs.Replace(bestgen))
	// 	} else {
	// 		AuthorGenre[aut] = bestgen
	// 	}
	// }
	// fmt.Printf("Authors-genres contents: %v\n\n", AuthorGenre)

	// const AuthorTempl = `{{define "Auth"}}{{range .Authors}}, {{.Lname}}{{if .Fname}} {{.Fname}}{{end}}{{if .Mname}} {{.Mname}}{{end}}{{end}}{{end}}`
	// const SeriesTempl = `{{define "Ser"}}{{if .Sequence}}{{.Sequence}}{{.SerSep}}{{if ge .SeqNo 1}}#{{.SeqNo}} - {{end}}{{end}}{{end}}`
	// const AuthorFolder = `{{.Origin}}_{{.Genre}}/{{template "Auth" .}}{{.AuthorSep}}{{template "Ser" .}}{{.Title}}{{.Ext}}`

	// t := template.New("Filename - author is folder")
	// t, err = t.Parse(AuthorFolder)
	// t, err = t.Parse(AuthorTempl)
	// t, err = t.Parse(SeriesTempl)

	// if err != nil {
	// 	fmt.Println("Fatal error ", err.Error())
	// 	os.Exit(1)
	// }

	// for _, b := range GoodBooks {
	// 	if genrepl, ok := AuthorGenre[b.Authors[0]]; ok {
	// 		b.Genres[0] = genrepl
	// 	}
	// 	r := strings.NewReplacer(
	// 		"/, ", "/",
	// 		"  ", " ")

	// 	type tempb struct {
	// 		AuthorSep, SerSep, Genre, Lname, Fname, Mname, Title, Ext, Origin, Sequence string
	// 		SeqNo                                                                       int
	// 		Authors                                                                     []Person
	// 	}

	// 	var NewFileName bytes.Buffer
	// 	BookMap := &tempb{}
	// 	BookMap.Genre = b.Genres[0]
	// 	BookMap.Authors = b.Authors
	// 	BookMap.Title = b.Title
	// 	BookMap.Ext = b.Ext
	// 	// Check is book belongs to any series
	// 	if len(b.AuthorSequences) > 0 {
	// 		BookMap.Sequence = b.AuthorSequences[0].Name
	// 		BookMap.SeqNo = b.AuthorSequences[0].Number
	// 	} else {
	// 		BookMap.Sequence = ""
	// 		BookMap.SeqNo = 0
	// 	}
	// 	if AuthorsBookCounter[b.Authors[0]] >= 3 {
	// 		BookMap.AuthorSep = "/"
	// 	} else {
	// 		BookMap.AuthorSep = " - "
	// 	}
	// 	if SequenceCounter[BookMap.Sequence] >= 3 || AuthorSequence[b.Authors[0]][BookMap.Sequence] > 1 {
	// 		BookMap.SerSep = "/"
	// 	} else {
	// 		BookMap.SerSep = " - "
	// 	}
	// 	n := 0

	// 	for lang, num := range AuthorOrigin[b.Authors[0]] {
	// 		if BookMap.Origin == "" || num > n {
	// 			BookMap.Origin = lang
	// 			n = num
	// 		}
	// 	}

	// 	fmt.Println(BookMap)
	// 	err = t.Execute(&NewFileName, BookMap)
	// 	if err != nil {
	// 		fmt.Println("Fatal error ", err.Error())
	// 		os.Exit(1)
	// 	}
	// 	nm := r.Replace(path.Clean(strings.Join([]string{*outdir, NewFileName.String()}, "/")))
	// 	fmt.Println("- New file name: ", nm)

	// 	if *rename {
	// 		fmt.Println("Renaming", b.Path, "to", nm)
	// 		os.MkdirAll(path.Dir(nm), 0777)
	// 		if err := os.Rename(b.Path, nm); err != nil {
	// 			fmt.Println("Rename error!", err)
	// 		}
	// 	}
	// }
	// //Initialize scroll bar and scan for files
	// // for _, boo := range books {
	// 	output, _ := xml.MarshalIndent(boo, "  ", "    ")
	// 	fmt.Printf("!-%s-\n", output)
}
