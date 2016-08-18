package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
	"text/template"

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
)

func findfiles(searchpath string) chan string {
	count := 0
	out := make(chan string)
	go func() {
		filepath.Walk(searchpath, func(path string, f os.FileInfo, _ error) error {
			if strings.HasSuffix(f.Name(), ".fb2") || strings.HasSuffix(f.Name(), ".zip") {
				out <- path
				count = count + 1
				// fmt.Print("\033[2K\033[0G")
				// fmt.Print("\033[2K")
				fmt.Printf("\rFiles: %d, queue len: %d [%s]", count, len(out), path)
			}
			return nil
		})
		defer close(out)
	}()
	return out
}

func worker(ID int, files chan string, out chan *Book, wg *sync.WaitGroup) { //, out chan *Book) {
	fmt.Printf("Worker %d launched\n", ID)
	defer wg.Done()
	count := 0
	for path := range files {
		out <- parsefb2(path)
		count = count + 1
		// fmt.Print("\033[2K\033[0G")
		fmt.Println("")
		fmt.Print("\r\033[2K")
		fmt.Printf("\rWorker %d: %d", ID, count)
	}
	fmt.Println("Worker ", ID, "is finished his job")
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
	gs := GenreSubstitutions{}
	gs.Create(conf)
	fmt.Println(gs)

	fmt.Println("Looking for files in ", *indir)
	FilesQueue := findfiles(*indir)

	wg := &sync.WaitGroup{}
	BooksQueue := make(chan *Book)
	for i := 0; i < 5; i++ {
		go worker(i, FilesQueue, BooksQueue, wg)
		wg.Add(1)
	}
	GoodBooks := []*Book{}
	ErrorBooks := []*Book{}
	authorscounter := make(AuthorsCounter)

	// Read all book files, sort correct and incorrect and accumulate authors info for list of corrections
	go func() {
		for b := range BooksQueue {
			if b.Error != nil {
				ErrorBooks = append(ErrorBooks, b)
				continue
			}

			GoodBooks = append(GoodBooks, b)
			for _, a := range b.Authors {
				authorscounter[a]++
			}
		}
	}()
	wg.Wait()

	// Create list of authors substitutions
	AuthorsReplaceList := GenerateAuthorReplace(authorscounter)

	type pg struct {
		a Person
		g string
	}
	AuthorsBookCounter := make(map[Person]int)
	SequenceCounter := map[string]int{}
	authgencounter := make(map[Person]map[string]int)
	AuthorOrigin := make(map[Person]map[string]int)
	AuthorSequence := make(map[Person]map[string]int)

	db, _ := OpenDB("")
	IDCount := 0
	NoIDConut := 0
	for _, b := range GoodBooks {
		if id := GetBookID(db, b); id > 0 {
			IDCount++
			fmt.Printf("Book title: %s, ID: %d, MD5: %s\n", b.Title, id, b.MD5)
			fmt.Println(GetAuthors(db, id))
		} else {
			fmt.Printf("No ID found for: %s\n", b.Path)
			fmt.Printf("%s: %s\n", b.Authors[0].Lname, b.Title)
			fmt.Println("-----------------------------")
			NoIDConut++
		}
	}
	fmt.Printf("Good books: %d, books with ISs: %d, noISs: %d", len(GoodBooks), IDCount, NoIDConut)
	return
	// !!!
	for _, b := range GoodBooks {
		for i, a := range b.Authors {
			if repl, ok := AuthorsReplaceList[a]; ok {
				b.Authors[i] = repl
			}
			AuthorsBookCounter[b.Authors[i]]++
			for _, g := range b.Genres {
				if g == "" {
					continue
				}
				if _, ok := authgencounter[b.Authors[i]]; !ok {
					authgencounter[b.Authors[i]] = make(map[string]int)
				}
				authgencounter[b.Authors[i]][g]++
			}

			if _, ok := AuthorOrigin[b.Authors[i]]; !ok {
				AuthorOrigin[b.Authors[i]] = make(map[string]int)
			}
			AuthorOrigin[b.Authors[i]][b.Translated()]++

			for _, s := range b.AuthorSequences {
				if _, ok := AuthorSequence[b.Authors[i]]; !ok {
					AuthorSequence[b.Authors[i]] = make(map[string]int)
				}
				AuthorSequence[b.Authors[i]][s.Name]++
			}
		}

		for _, s := range b.AuthorSequences {
			SequenceCounter[s.Name]++
		}

	}

	for seq, num := range SequenceCounter {
		if num > 10 {
			fmt.Println("S: ", seq, num)
		}
	}
	fmt.Printf("Found %d books and %d books with errors\n", len(GoodBooks), len(ErrorBooks))

	// jsonExport(AuthorsReplaceList, "a.json")

	// db, _ := CreateDB("")
	// for _, b := range GoodBooks {
	// 	AddBookDB(db, b)
	// }

	AuthorGenre := make(map[Person]string)

	for aut, gen := range authgencounter {
		bestgen := ""
		count := 0
		for g, c := range gen {
			if c > count {
				bestgen = g
				count = c
			}
		}
		if gs.Replace(bestgen) != "" {
			AuthorGenre[aut] = strings.ToLower(gs.Replace(bestgen))
		} else {
			AuthorGenre[aut] = bestgen
		}
	}
	fmt.Printf("Authors-genres contents: %v\n\n", AuthorGenre)

	const AuthorTempl = `{{define "Auth"}}{{range .Authors}}, {{.Lname}}{{if .Fname}} {{.Fname}}{{end}}{{if .Mname}} {{.Mname}}{{end}}{{end}}{{end}}`
	const SeriesTempl = `{{define "Ser"}}{{if .Sequence}}{{.Sequence}}{{.SerSep}}{{if ge .SeqNo 1}}#{{.SeqNo}} - {{end}}{{end}}{{end}}`
	const AuthorFolder = `{{.Origin}}_{{.Genre}}/{{template "Auth" .}}{{.AuthorSep}}{{template "Ser" .}}{{.Title}}{{.Ext}}`

	t := template.New("Filename - author is folder")
	t, err = t.Parse(AuthorFolder)
	t, err = t.Parse(AuthorTempl)
	t, err = t.Parse(SeriesTempl)

	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}

	for _, b := range GoodBooks {
		if genrepl, ok := AuthorGenre[b.Authors[0]]; ok {
			b.Genres[0] = genrepl
		}
		r := strings.NewReplacer(
			"/, ", "/",
			"  ", " ")

		type tempb struct {
			AuthorSep, SerSep, Genre, Lname, Fname, Mname, Title, Ext, Origin, Sequence string
			SeqNo                                                                       int
			Authors                                                                     []Person
		}

		var NewFileName bytes.Buffer
		BookMap := &tempb{}
		BookMap.Genre = b.Genres[0]
		BookMap.Authors = b.Authors
		BookMap.Title = b.Title
		BookMap.Ext = b.Ext
		// Check is book belongs to any series
		if len(b.AuthorSequences) > 0 {
			BookMap.Sequence = b.AuthorSequences[0].Name
			BookMap.SeqNo = b.AuthorSequences[0].Number
		} else {
			BookMap.Sequence = ""
			BookMap.SeqNo = 0
		}
		if AuthorsBookCounter[b.Authors[0]] >= 3 {
			BookMap.AuthorSep = "/"
		} else {
			BookMap.AuthorSep = " - "
		}
		if SequenceCounter[BookMap.Sequence] >= 3 || AuthorSequence[b.Authors[0]][BookMap.Sequence] > 1 {
			BookMap.SerSep = "/"
		} else {
			BookMap.SerSep = " - "
		}
		n := 0

		for lang, num := range AuthorOrigin[b.Authors[0]] {
			if BookMap.Origin == "" || num > n {
				BookMap.Origin = lang
				n = num
			}
		}

		fmt.Println(BookMap)
		err = t.Execute(&NewFileName, BookMap)
		if err != nil {
			fmt.Println("Fatal error ", err.Error())
			os.Exit(1)
		}
		nm := r.Replace(path.Clean(strings.Join([]string{*outdir, NewFileName.String()}, "/")))
		fmt.Println("- New file name: ", nm)

		if *rename {
			fmt.Println("Renaming", b.Path, "to", nm)
			os.MkdirAll(path.Dir(nm), 0777)
			if err := os.Rename(b.Path, nm); err != nil {
				fmt.Println("Rename error!", err)
			}
		}
	}
	// //Initialize scroll bar and scan for files
	// // for _, boo := range books {
	// 	output, _ := xml.MarshalIndent(boo, "  ", "    ")
	// 	fmt.Printf("!-%s-\n", output)
}
