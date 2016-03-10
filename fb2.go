package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"

	_ "github.com/mattn/go-sqlite3"
)

var (
	root            = flag.String("r", ".", "Path to unsorted books")
	checkauthors    = flag.Bool("A", false, "Scan books for authors and generate lists of corrections")
	authcompilation = flag.Int("c", 2, "Number of authors is book file to set author to compilaton")
	rename          = flag.Bool("R", false, "Rename founded files")
	configfile      = flag.String("c", "config.yaml", "Path to config file")
)

func findfiles(searchpath string, out chan string) {
	go func() {
		filepath.Walk(searchpath, func(path string, f os.FileInfo, _ error) error {
			if strings.HasSuffix(f.Name(), ".fb2") || strings.HasSuffix(f.Name(), ".zip") {
				out <- path
			}
			return nil
		})
		defer close(out)
	}()
}

// bar := pb.StartNew(len(filesToProcess))
func worker(ID int, files chan string, out chan *Book, wg *sync.WaitGroup) { //, out chan *Book) {
	fmt.Printf("Worker %d launched\n", ID)
	defer wg.Done()
	for path := range files {
		out <- parsefb2(path)
	}
	fmt.Println("Worker ", ID, "is finished his job")

	// processed += 1
	// }
	// bar.Increment()
	// }
	// wg.Done()
}

func main() {

	flag.Parse()
	// conf, err := readConfig(*configfile)
	FilesQueue := make(chan string)
	BooksQueue := make(chan *Book)
	fmt.Println("Looking for files in ", *root)

	wg := &sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		go worker(i, FilesQueue, BooksQueue, wg)
		wg.Add(1)
	}
	findfiles(*root, FilesQueue)
	GoodBooks := []*Book{}
	ErrorBooks := []*Book{}

	a := make(AuthorGroups)
	ag := &a
	seqlist := map[string]int{}

	gs := GenreSubstitutions{}
	gs.Create()

	type pg struct {
		a Person
		g string
	}

	go func() {
		for b := range BooksQueue {
			if b.Error != nil {
				ErrorBooks = append(ErrorBooks, b)
			} else {
				GoodBooks = append(GoodBooks, b)
				fmt.Println(b.Title)
				for _, s := range b.Sequences {
					seqlist[s.Name]++
				}
			}
		}
	}()
	wg.Wait()

	for seq, num := range seqlist {
		if num > 10 {
			fmt.Println("S: ", seq, num)
		}
	}
	for _, book := range ErrorBooks {
		fmt.Printf("Error in book: %s - %s\n", path.Base(book.Path), book.Error)
	}
	fmt.Printf("Found %d books and %d books with errors\n", len(GoodBooks), len(ErrorBooks))

	// Create list of authors substitutions
	AuthorsReplaceList := make(map[string]Person)
	AuthorsBookCounter := make(map[string]int)
	if *checkauthors {
		for _, b := range GoodBooks {
			for _, a := range b.Authors {
				ag.Add(a)
				AuthorsBookCounter[a.Fingerprint()]++
			}
		}
		for _, k := range *ag {
			if len(k) < 2 {
				continue
			}
			sort.Sort(ByLength{k})
			for i, a := range k {
				if i > 0 && a.Author.String() != k[0].Author.String() {
					AuthorsReplaceList[a.Author.String()] = k[0].Author
				}
			}
		}
		for from, to := range AuthorsReplaceList {
			fmt.Printf("Replace %v with %v\n", from, to)
		}
	}

	authgencounter := make(map[Person]map[string]int)
	authorscounter := make(map[string]int)

	for _, b := range GoodBooks {
		for i, a := range b.Authors {
			if repl, ok := AuthorsReplaceList[a.String()]; ok {
				b.Authors[i] = repl
			}
			authorscounter[b.Authors[i].String()]++
		}
		for _, a := range b.Authors {
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
	jsonExport(AuthorsReplaceList, "a.json")

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
	fmt.Printf("Authors replace contents: %v\n\n", ag)
	fmt.Printf("Authors-genres contents: %v\n\n", AuthorGenre)

	const AuthorFlat = `{{define Author}}{{if .Lname}}{{ .Lname }}{{end}} {{if .Fname}} {{ .Fname }}{{end}} {{if .Mname}} {{ .Mname }}{{end}}{{end}}`
	const AuthorFolder = `{{define Author}}/{{if .Lname}}{{ .Lname }}{{end}} {{if .Fname}} {{ .Fname }}{{end}} {{if .Mname}} {{ .Mname }}{{end}}/{{end}}`

	const templ = `
	{{define "Author"}}{{if .Lname}}{{ .Lname }}{{end}}{{if .Fname}} {{ .Fname }}{{end}}{{if .Mname}} {{ .Mname }}{{end}}{{end}}
	{{define "SeqNo"}}{{if .Number}}{{.Number}}{{else}}{{end}}{{end}}
	{{define "Seq"}}{{if .Sequences }}{{range .Sequences}}{{.Name}}{{end}}{{end}}
	
	./{{index .Genres 0}} /{{template "Author" index .Authors 0 }}/{{template "Seq" .Sequences}}/{{template "SeqNo" .Sequences}} - {{.Title}}{{.Ext}}
	`
	const flatauthor = `
	{{define "Author"}}{{if .Lname}}{{ .Lname }}{{end}}{{if .Fname}} {{ .Fname }}{{end}}{{if .Mname}} {{ .Mname }}{{end}}{{end}}
	{{define "SeqNo"}}{{if .Number}}{{.Number}}{{end}}{{end}}
	{{define "Seq"}}{{if .Sequences }}{{range .Sequences}}{{.Name}}{{end}}{{end}}{{end}}
	
	./{{index .Genres 0}} /{{template "Author" index .Authors 0 }} - {{template "Seq" .Sequences}} - #{{template "SeqNo" .Sequences}} - {{.Title}}{{.Ext}}
	`

	t := template.Must(template.New("").Parse(templ))
	f := template.Must(template.New("").Parse(flatauthor))
	// t = template.New(templ).Funcs(defaultfuncs)
	// t, _ = t.Parse(subtmpl)
	// t, _ = t.Parse(hometmpl)

	for _, b := range GoodBooks {
		fmt.Println("1")
		if genrepl, ok := AuthorGenre[b.Authors[0]]; ok {
			b.Genres[0] = genrepl
		}

		var buf bytes.Buffer
		if authorscounter[b.Authors[0].String()] > 4 {
			t.Execute(&buf, b)
		} else {
			f.Execute(&buf, b)
		}
		fmt.Println("2")
		r := strings.NewReplacer("\n", "", "\t", "", "  ", " ")
		nm := genName(strings.TrimSpace(path.Clean(r.Replace(buf.String()))))
		fmt.Println("4")

		if *rename {
			fmt.Println("Renaming", b.Path, "to", nm)
			fmt.Println("5")
			os.MkdirAll(path.Dir(nm), 0777)
			fmt.Println("6")
			if err := os.Rename(b.Path, nm); err != nil {
				fmt.Println("7")
				fmt.Println("Rename error!", err)
			}
			fmt.Println("8")
		}
		fmt.Println("9")
	}
	// //Initialize scroll bar and scan for files
	// // for _, boo := range books {
	// 	output, _ := xml.MarshalIndent(boo, "  ", "    ")
	// 	fmt.Printf("!-%s-\n", output)
	// }
	// fmt.Printf("Files %d/%d processed, %d errors\n", len(filesToProcess), len(books), processErrors)
}
