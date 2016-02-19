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

func findfiles(searchpath string, out chan string) {
	go func() {
		filepath.Walk(searchpath, func(path string, f os.FileInfo, _ error) error {
			if strings.HasSuffix(f.Name(), ".fb2") || strings.HasSuffix(f.Name(), ".fb2.zip") {
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
	root := flag.String("r", ".", "Path to unsorted books")
	checkauthors := flag.Bool("A", false, "Scan books for authors and generate lists of corrections")
	//rename := flag.Bool("R", false, "Rename founded files")

	flag.Parse()

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

	AuthorsReplaceList := make(map[Person]Person)
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
	if *checkauthors {
		for _, b := range GoodBooks {
			for _, author := range b.Authors {
				ag.Add(author)
			}
		}
		for _, k := range *ag {
			if len(k) < 2 {
				continue
			}
			sort.Sort(ByLength{k})
			for i, a := range k {
				if i > 0 {
					AuthorsReplaceList[a.Author] = k[0].Author
				}
			}
		}
		for from, to := range AuthorsReplaceList {
			fmt.Printf("Replace %v with %v\n", from, to)
		}
	}

	authgencounter := make(map[Person]map[string]int)

	for _, b := range GoodBooks {
		for i, a := range b.Authors {
			if repl, ok := AuthorsReplaceList[a]; ok {
				b.Authors[i] = repl
			}
		}
		for _, a := range b.Authors {
			for _, g := range b.Genres {
				if _, ok := authgencounter[a]; !ok {
					authgencounter[a] = make(map[string]int)
				}
				authgencounter[a][g]++
			}
		}
	}

	// db, _ := CreateDB("")
	// for _, b := range GoodBooks {
	// 	AddBookDB(db, b)
	// }

	fmt.Println(authgencounter)

	a_g := make(map[Person]string)

	for aut, gen := range authgencounter {
		bestgen := ""
		count := 0
		for g, c := range gen {
			if c > count {
				bestgen = g
				count = c
			}
		}
		a_g[aut] = bestgen
	}

	fmt.Println(a_g)

	const templ = `
	{{define "Author"}}
		{{if .Lname}}{{ .Lname }}{{end}}
		{{if .Fname}} {{ .Fname }}{{end}}
		{{if .Mname}} {{ .Mname }}{{end}}
	{{end}}
	/{{index .Genres 0}}
	/{{template "Author" index .Authors 0 }}
	/{{if .Sequences }}{{index .Sequences 0}}{{end}}
	/{{.Title}}
	`
	t := template.Must(template.New("").Parse(templ))

	for _, b := range GoodBooks {
		if genrepl, ok := a_g[b.Authors[0]]; ok {
			b.Genres[0] = genrepl
		}

		var buf bytes.Buffer
		t.Execute(&buf, b)
		r := strings.NewReplacer("\n", "", "\t", "")
		fmt.Println(strings.TrimSpace(path.Clean(r.Replace(buf.String()))))
	}
	// //Initialize scroll bar and scan for files
	// // for _, boo := range books {
	// 	output, _ := xml.MarshalIndent(boo, "  ", "    ")
	// 	fmt.Printf("!-%s-\n", output)
	// }
	// fmt.Printf("Files %d/%d processed, %d errors\n", len(filesToProcess), len(books), processErrors)
}
