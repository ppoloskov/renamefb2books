package main

import (
	"log"
	"regexp"
	"sort"
	"strings"
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
	return strings.TrimSpace(strings.Join([]string{author.Lname, author.Fname, author.Mname}, " "))
}
