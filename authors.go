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

type Counters []*Counter

type AuthorGroups map[string]Counters

// Len is part of sort.Interface.
func (a Counters) Len() int      { return len(a) }
func (a Counters) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByNumber struct{ Counters }
type ByLength struct{ Counters }

// func ByNumeber(items []Item) byWeight {
// append creates a copy of items, they are appended to the empty slice byWeight{}
// 	return append(byWeight{}, items...)
// }

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
	// Prefer authors with middlenames. Drop i if no middle name
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

func (ag *AuthorGroups) Add(author Person) {
	ind := author.Fingerprint()

	// if group exists we look it throw to check if author is in it. If so - increment counter and return
	if len((*ag)[ind]) > 0 {
		for _, k := range (*ag)[ind] {
			//if strings.Compare(k.Author.Lname, author.Lname) == 0 &&
			//	strings.Compare(k.Author.Fname, author.Fname) == 0 {
			//	k.Author.Mname == author.Mname {
			if reflect.DeepEqual(k.Author, author) {
				k.Count++
				return
			}
		}
	}
	// In other cases we add author to group
	(*ag)[ind] = append((*ag)[ind], &Counter{Author: author, Count: 1})
}
