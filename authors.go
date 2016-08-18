package main

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type Person struct {
	LRSId int
	Fname string `xml:"first-name"`
	Mname string `xml:"middle-name"`
	Lname string `xml:"last-name"`
	Nick  string `xml:"nickname"`
	Email string `xml:"email"`
	Id    string `xml:"id"`
}

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
	reg, err := regexp.Compile("[^a-zа-я]+")
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

func (author Person) String() string {
	return fmt.Sprintf("L:\"%s\", F:\"%s\", M:\"%s\"", author.Lname, author.Fname, author.Mname)
}
func (author Person) LongAuthorString() string {
	cleanString := func(r rune) rune {
		if unicode.IsLetter(r) {
			return r
		} else {
			return -1
		}
	}

	return strings.Map(cleanString, strings.Join([]string{author.Lname, author.Fname, author.Mname}, " "))
}

type Counter struct {
	Author Person
	Count  int
}

type Counters []Counter
type AuthorsCounter map[Person]int

// Len is part of sort.Interface.
func (a Counters) Len() int      { return len(a) }
func (a Counters) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByNumber struct{ Counters }
type ByLength struct{ Counters }

// Decresing sort > is reversed
func (a ByNumber) Less(i, j int) bool {
	return (a.Counters[i].Count > a.Counters[j].Count)
}

func BegEndSpace(s string) bool {
	if len(s) > len(strings.TrimSpace(s)) {
		return true
	} else {
		return false
	}
}

func (a ByLength) Less(i, j int) bool {
	// Prefer authors with middlenames. Drop if no middle name
	if len(a.Counters[i].Author.Mname) < len(a.Counters[j].Author.Mname) {
		return false
	}
	// Prefer authors without spaces at the beginning or end of namei. Drop i if contains spaces.
	if BegEndSpace(a.Counters[i].Author.Lname) || BegEndSpace(a.Counters[i].Author.Fname) || BegEndSpace(a.Counters[i].Author.Mname) {
		return false
	}
	// Prefer authors without spaces at the beginning or end of namei. Drop j if contains spaces.
	if BegEndSpace(a.Counters[j].Author.Lname) || BegEndSpace(a.Counters[j].Author.Fname) || BegEndSpace(a.Counters[j].Author.Mname) {
		return true
	}
	// author i is better
	return (len(a.Counters[i].Author.LongAuthorString()) > len(a.Counters[j].Author.LongAuthorString()))
}

func GenerateAuthorReplace(authorscounter AuthorsCounter) map[Person]Person {
	ag := make(map[string]Counters)
	AuthorsReplaceList := make(map[Person]Person)

	for author, count := range authorscounter {
		ind := author.Fingerprint()
		// if group exists we look it throw to check if author is in it. If so - increment counter and return
		// In other cases we add author to group
		if len(ag[ind]) == 0 {
			ag[ind] = append(ag[ind], Counter{Author: author, Count: count})
			continue
		}

		for _, k := range ag[ind] {
			if reflect.DeepEqual(k.Author, author) {
				k.Count += count
				continue
			}
		}
		ag[ind] = append(ag[ind], Counter{Author: author, Count: count})
	}
	for _, k := range ag {
		if len(k) == 1 {
			continue
		}
		sort.Sort(ByLength{k})
		fmt.Println(k)
		for i := 1; i < len(k); i++ {
			if k[i].Author.String() != k[0].Author.String() {
				AuthorsReplaceList[k[i].Author] = k[0].Author
			}
		}
	}
	fmt.Println("Author correntions:")
	for from, to := range AuthorsReplaceList {
		fmt.Printf("Replace %v with %v\n", from, to)
	}
	return AuthorsReplaceList
}
