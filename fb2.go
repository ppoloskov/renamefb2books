package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	_ "github.com/mattn/go-sqlite3"
	// "github.com/metakeule/places"
)

var (
	root            = flag.String("r", "", "Path to unsorted books")
	checkauthors    = flag.Bool("A", false, "Scan books for authors and generate lists of corrections")
	authcompilation = flag.Int("comp", 2, "Number of authors is book file to set author to compilaton")
	rename          = flag.Bool("R", false, "Rename founded files")
	configfile      = flag.String("c", "config.yaml", "Path to config file")
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

	conf, err := readConfig(*configfile)
	if err != nil {
		panic(err)
	}
	gs := GenreSubstitutions{}
	gs.Create(conf)
	fmt.Println(gs)

	fmt.Println("Looking for files in ", *root)
	FilesQueue := findfiles(*root)

	wg := &sync.WaitGroup{}
	BooksQueue := make(chan *Book)
	for i := 0; i < 5; i++ {
		go worker(i, FilesQueue, BooksQueue, wg)
		wg.Add(1)
	}
	GoodBooks := []*Book{}
	ErrorBooks := []*Book{}

	type pg struct {
		a Person
		g string
	}

	SequenceCounter := map[string]int{}
	AuthorsBookCounter := make(map[string]int)
	authgencounter := make(map[Person]map[string]int)
	authorscounter := make(AuthorsCounter)

	go func() {
		for b := range BooksQueue {
			if b.Error != nil {
				ErrorBooks = append(ErrorBooks, b)
				continue
			}

			GoodBooks = append(GoodBooks, b)
			for _, s := range b.Sequences {
				SequenceCounter[s.Name]++
			}
			for _, a := range b.Authors {
				AuthorsBookCounter[a.Fingerprint()]++
				authorscounter[a]++

				for _, g := range b.Genres {
					if g == "" {
						continue
					}
					if _, ok := authgencounter[a]; !ok {
						authgencounter[a] = make(map[string]int)
					}
					authgencounter[a][g]++
				}
			}
		}
	}()
	fmt.Println("Wait")
	wg.Wait()

	for seq, num := range SequenceCounter {
		if num > 10 {
			fmt.Println("S: ", seq, num)
		}
	}
	// for _, book := range ErrorBooks {
	// 	fmt.Printf("Error in book: %s - %s\n", path.Base(book.Path), book.Error)
	// }
	fmt.Printf("Found %d books and %d books with errors\n", len(GoodBooks), len(ErrorBooks))

	// Create list of authors substitutions
	AuthorsReplaceList := GenerateAuthorReplace(authorscounter)

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

	const AuthorTempl = `{{define "A"}}{{.Lname}}{{if .Fname}} {{.Fname}}{{end}}{{if .Mname}} {{.Mname}}{{end}}{{end}}`
	const AuthorFolder = `{{.Genre}}/{{template "A" .}}/{{if .Sequence}}{{.Sequence}} - {{end}}{{if .SeqNo}}#{{.SeqNo}} - {{end}}{{.Title}}{{.Ext}}`
	const AuthorFlat = `{{.Genre}}/{{template "A" .}} - {{if .Sequence}}{{.Sequence}} - {{end}}{{if .SeqNo}}#{{.SeqNo}} - {{end}}{{.Title}}{{.Ext}}`

	t := template.New("Filename - author is folder")
	t, err = t.Parse(AuthorTempl)
	t, err = t.Parse(AuthorFolder)

	// f := template.New("Filename - author is part of filename")
	// f, err = f.Parse(AuthorFlat)
	// f, err = f.Parse(AuthorTempl)

	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}

	for _, b := range GoodBooks {
		for i, a := range b.Authors {
			if repl, ok := AuthorsReplaceList[a]; ok {
				b.Authors[i] = repl
			}
		}
		if genrepl, ok := AuthorGenre[b.Authors[0]]; ok {
			b.Genres[0] = genrepl
		}
		// r := strings.NewReplacer(" /", "/", "/ -", "/", " # -", "", "\n", "", "\t", "", "  ", " ")
		//r := strings.NewReplacer("/ -", "/", "# -", "")
		type tempb struct {
			Genre, Lname, Fname, Mname, Title, Ext, Sequence, SeqNo string
		}

		var NewFileName bytes.Buffer
		BookMap := &tempb{}
		BookMap.Genre = b.Genres[0]
		BookMap.Lname = b.Authors[0].Lname
		BookMap.Fname = b.Authors[0].Fname
		BookMap.Mname = b.Authors[0].Mname
		BookMap.Title = b.Title
		BookMap.Ext = b.Ext

		if len(b.Sequences) >= 1 {
			BookMap.Sequence = b.Sequences[0].Name
			BookMap.SeqNo = fmt.Sprintf("%d", b.Sequences[0].Number)
		} else {
			BookMap.Sequence = ""
			BookMap.SeqNo = ""
		}

		// if authorscounter[b.Authors[0].String()] >= 3 {
		err = t.Execute(&NewFileName, BookMap)
		// } else {
		// 	err = f.Execute(&NewFileName, BookMap)
		// }

		if err != nil {
			fmt.Println("Fatal error ", err.Error())
			os.Exit(1)
		}
		fmt.Println("- New file name: ", path.Clean(NewFileName.String()))

		// if *rename {
		// 	fmt.Println("Renaming", b.Path, "to", nm)
		// 	fmt.Println("5")
		// 	os.MkdirAll(path.Dir(nm), 0777)
		// 	fmt.Println("6")
		// 	if err := os.Rename(b.Path, nm); err != nil {
		// 		fmt.Println("7")
		// 		fmt.Println("Rename error!", err)
		// 	}
		// 	fmt.Println("8")
		// }
	}
	// //Initialize scroll bar and scan for files
	// // for _, boo := range books {
	// 	output, _ := xml.MarshalIndent(boo, "  ", "    ")
	// 	fmt.Printf("!-%s-\n", output)
	// }
	// fmt.Printf("Files %d/%d processed, %d errors\n", len(filesToProcess), len(books), processErrors)
}
