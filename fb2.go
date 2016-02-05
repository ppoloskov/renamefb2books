package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/ryanuber/go-glob"
)

// Len is part of sort.Interface.
func (a acount) Len() int {
	return len(a)
}

// Swap is part of sort.Interface.
func (a acount) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Less is part of sort.Interface. We use count as the value to sort by
func (a acount) Less(i, j int) bool {
	return (a[i].Counter < a[j].Counter)
}

func findfiles(searchpath string, out chan string) {
	go func() {
		filepath.Walk(searchpath, func(path string, f os.FileInfo, _ error) error {
			if matched, _ := filepath.Match("*.fb2", f.Name()); matched {
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

func repl(b Book) {

	var DirectMatch = make(map[string]string)
	var WildMatch = make(map[string]string)

	for _, p := range Patterns {
		for _, mt := range p.From {
			if strings.ContainsAny(mt, "*") {
				WildMatch[mt] = p.To
			} else {
				DirectMatch[mt] = p.To
			}
		}

	}
	gen := ""
	ok := false
	if len(b.Genres) > 0 {
		for _, bookgenre := range b.Genres {
			if len(bookgenre) == 0 {
				continue
			}
			gen, ok = DirectMatch[bookgenre]
			if ok {
				break
			}
			for wild := range WildMatch {
				if glob.Glob(wild, bookgenre) {
					gen = WildMatch[wild]
					ok = true
					break
				}
			}
			if ok {
				break
			}
		}
	}
	if gen == "" {
		fmt.Printf("Just got: '%s - %s'\n", b.Title, b.Genres)
	} else {
		fmt.Printf("Just got: '%s - %s'\n", b.Title, gen)
	}
}

type Key struct {
	Author  Person
	Counter int
}

type acount []Key

func NormalizeText(s string) string {
	words := strings.Fields(s)
	smallwords := " a an on the to в на или не х"

	r := strings.NewReplacer("Ё", "Е", ">", "&gt;")
	fmt.Println(r.Replace("This is <b>HTML</b>!"))
	// !"'()+,-.:;=[\]{}«»Ёё–—
	for index, word := range words {
		if strings.Contains(smallwords, " "+word+" ") {
			words[index] = word
		} else {
			words[index] = strings.Title(word)
		}
	}
	return strings.Join(words, " ")
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
	// alist := authorSlice{}
	AuthorsCounter := make(map[string]acount)
	seqlist := map[string]int{}

	go func() {
		for b := range BooksQueue {
			if b.Error != nil {
				ErrorBooks = append(ErrorBooks, b)
			} else {
				GoodBooks = append(GoodBooks, b)

				for _, author := range b.Authors {
					countindex := author.Fingerprint()
					if len(AuthorsCounter[countindex]) > 0 {
						ok := false
						for i, k := range AuthorsCounter[countindex] {
							//if strings.Compare(k.Author.Lname, author.Lname) == 0 &&
							//	strings.Compare(k.Author.Fname, author.Fname) == 0 {
							//	k.Author.Mname == author.Mname {
							if reflect.DeepEqual(k.Author, author) {
								AuthorsCounter[countindex][i].Counter++
								ok = true
								break
							}
						}
						if !ok {
							AuthorsCounter[countindex] = append(AuthorsCounter[countindex], Key{author, 1})
						}

					} else {
						AuthorsCounter[countindex] = append(AuthorsCounter[countindex], Key{author, 1})
					}
				}

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

	if *checkauthors {
		for _, k := range AuthorsCounter {
			if len(k) < 2 {
				continue
			}
			sort.Sort(k)
			for i, _ := range k {
				if i < len(k)-1 {
					fmt.Printf("Replace '%q' with '%q'\n", k[i].Author, k[len(k)-1].Author)
				}
			}
		}
		jsonExport(seqlist, "s.json")
		jsonExport(AuthorsCounter, "a.json")
	}
	// //Initialize scroll bar and scan for files
	// // for _, boo := range books {
	// 	output, _ := xml.MarshalIndent(boo, "  ", "    ")
	// 	fmt.Printf("!-%s-\n", output)
	// }
	// fmt.Printf("Files %d/%d processed, %d errors\n", len(filesToProcess), len(books), processErrors)
}
