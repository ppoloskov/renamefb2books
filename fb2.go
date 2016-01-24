package main

import (
	"flag"
	"fmt"
	"github.com/ryanuber/go-glob"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"
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
	for i, _ := range arr {
		n := string(arr[i])
		if strings.TrimSpace(n) != "" {
			out = append(out, n)
		}
	}
	return strings.Join(out, " ")
}

func fingerprint(author Person) string {
	// bgram := map[string]struct{}{}
	// myString := author.Lname + author.Fname + author.Mname
	// var arr []string

	// for _, value := range myString {
	// 	if unicode.IsLetter(value) {
	// 		arr = append(arr, strings.ToLower(string(value)))
	// 	}
	// }
	// for i := 0; i < len(arr)-1; i++ {
	// 	bgram[strings.Join(arr[i:i+2], "")] = struct{}{}
	// }
	// list := []string{}
	// for i, _ := range bgram {
	// 	list = append(list, string(i))
	// }
	// sort.Strings(list)
	// return strings.Join(list, "")
	bgram := map[string]struct{}{}
	myString := author.Lname + author.Fname

	for _, value := range myString {
		if unicode.IsLetter(value) {
			bgram[strings.ToLower(string(value))] = struct{}{}
		}
	}
	list := []string{}
	for i, _ := range bgram {
		list = append(list, string(i))
	}
	sort.Strings(list)
	return strings.Join(list, "")
}

func opendb() {
	return
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

func main() {
	flag.Parse()
	root := flag.Arg(0)
	FilesQueue := make(chan string)
	BooksQueue := make(chan *Book)
	if root == "" {
		usr, _ := user.Current()
		root = usr.HomeDir
	}
	fmt.Println("Looking for files in ", root)

	wg := &sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		go worker(i, FilesQueue, BooksQueue, wg)
		wg.Add(1)
	}
	findfiles(root, FilesQueue)
	GoodBooks := []*Book{}
	ErrorBooks := []*Book{}
	// alist := authorSlice{}
	authorslist := map[string]int{}
	seqlist := map[string]int{}

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

	go func() {
		for b := range BooksQueue {
			if b.Error != nil {
				ErrorBooks = append(ErrorBooks, b)
			} else {
				GoodBooks = append(GoodBooks, b)
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
				for _, author := range b.Authors {
					authorslist[author.ToString()]++
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
	// //Initialize scroll bar and scan for files
	// // for _, boo := range books {
	// 	output, _ := xml.MarshalIndent(boo, "  ", "    ")
	// 	fmt.Printf("!-%s-\n", output)
	// }
	// fmt.Printf("Files %d/%d processed, %d errors\n", len(filesToProcess), len(books), processErrors)
}
