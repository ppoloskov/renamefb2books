package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/ryanuber/go-glob"
)

type authorSlice []Person

// Len is part of sort.Interface.
func (a authorSlice) Len() int {
	return len(a)
}

// Swap is part of sort.Interface.
func (a authorSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Less is part of sort.Interface. We use count as the value to sort by
func (a authorSlice) Less(i, j int) bool {
	return (len(a[i].Fname) + len(a[i].Mname) + len(a[i].Lname)) < (len(a[j].Fname) + len(a[j].Mname) + len(a[j].Lname))
}

func (a authorSlice) Exists(Author Person) bool {
	for _, aut := range a {
		if aut.Fname == Author.Fname && aut.Lname == Author.Lname {
			return true
		}
	}
	return false
}

// var alist = make(map[string]*Author)
// var a = make(map[Author]struct{})

func NormalizeSpaces(arr string) string {
	out := []string{}
	for i := range arr {
		n := string(arr[i])
		if strings.TrimSpace(n) != "" {
			out = append(out, n)
		}
	}
	return strings.Join(out, " ")
}

func (author Person) Fingerprint() string {
	// List of unique B-Grams
	bgram := make(map[string]bool)
	reg, err := regexp.Compile("[^A-Za-zА-Яа-я]+")
	if err != nil {
		log.Fatal(err)
	}

	myStringRunes := []rune(reg.ReplaceAllString(strings.ToLower(author.Lname+author.Fname), ""))

	for i, j := 0, 2; j < len(myStringRunes); i, j = i+1, j+1 {
		bgram[string(myStringRunes[i:j])] = true
	}
	list := []string{}
	for i, _ := range bgram {
		list = append(list, string(i))
	}
	sort.Strings(list)
	return strings.Join(list, "")
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
	AuthorsCounter := map[string][]Person{}
	seqlist := map[string]int{}

	go func() {
		for b := range BooksQueue {
			if b.Error != nil {
				ErrorBooks = append(ErrorBooks, b)
			} else {
				GoodBooks = append(GoodBooks, b)
				for _, author := range b.Authors {
					fmt.Println(author.Fname + " " + author.Lname)
					AuthorsCounter[author.Fingerprint()] = append(AuthorsCounter[author.Fingerprint()], author)
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
		jsonExport(AuthorsCounter, "a.json")
	}
	// //Initialize scroll bar and scan for files
	// // for _, boo := range books {
	// 	output, _ := xml.MarshalIndent(boo, "  ", "    ")
	// 	fmt.Printf("!-%s-\n", output)
	// }
	// fmt.Printf("Files %d/%d processed, %d errors\n", len(filesToProcess), len(books), processErrors)
}
