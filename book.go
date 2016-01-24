package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"code.google.com/p/go-charset/charset"
	_ "code.google.com/p/go-charset/data"
)

type Book struct {
	Error     error
	Path      string
	XMLName   xml.Name   `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 FictionBook"`
	Title     string     `xml:"description>title-info>book-title"`
	Authors   []Person   `xml:"description>title-info>author"`
	Genres    []string   `xml:"description>title-info>genre"`
	Sequences []Sequence `xml:"description>title-info>sequence"`
	SrcLang   string     `xml:"description>title-info>src-lang"`
	Keywords  []string   `xml:"description>title-info>keywords"`
	Date      string     `xml:"description>title-info>date"`
	Coverpage string     `xml:"description>title-info>coverpage"`
	//	 Annotation  string     `xml:"description>title-info>annotation"`
	Annotation struct {
		Items []struct {
			XMLName xml.Name
			Content string `xml:",innerxml"`
		} `xml:",any"`
	} `xml:"description>title-info>annotation"`
	Lang        string   `xml:"description>title-info>lang"`
	Translators []Person `xml:"description>title-info>translator"`
	// Translated  bool
	rest []string `xml:,any`
}

// // <description> - который описывает заголовок документа. Одно и только одно вхождение. (фразы вроде "одно и только одно вхождение" говорят, сколько раз подряд может идти данный тэг в данном месте документа)
// <body> - описывает тело документа. Одно или более вхождений.
// <binary>

// <src-title-info> - данные об исходнике книги (до перевода). От нуля до одного вхождений.
// <document-info> - информация об FB2-документе. Одно и только одно вхождение.
// <publish-info> - сведения об издании книги, которая была использована как источник при подготовке документа. От нуля до одного вхождений.
// <custom-info>

var Translated = func(b Book) bool {
	if strings.ToLower(b.SrcLang) == "ru" || b.SrcLang == "" {
		return false
	} else {
		return true
	}
}

type Person struct {
	Fname string `xml:"first-name"`
	Mname string `xml:"middle-name"`
	Lname string `xml:"last-name"`
	Nick  string `xml:"nickname"`
	Email string `xml:"email"`
	Id    string `xml:"id"`
}

type Sequence struct {
	Name   string `xml:"name,attr"`
	Number int    `xml:"number,attr"`
}

func (p *Person) ToString() string {
	return p.Lname + p.Fname
}

func StringIsUpper(s string) bool {
	for i := 0; i < len(s); {
		if i < len(s) {
			r, w := utf8.DecodeRuneInString(s[i:])
			i += w
			if unicode.IsLetter(r) && unicode.IsUpper(r) == false {
				return false
			}
		}
	}
	return true
}

func parsefb2(path string) *Book {
	b := Book{}
	b.Error = nil
	b.Path = path

	xmlFile, err := os.Open(path)
	if err != nil {
		fmt.Println("Error opening file:", err)
		b.Error = err
		return &b
	}
	defer xmlFile.Close()

	decoder := xml.NewDecoder(xmlFile)
	decoder.CharsetReader = charset.NewReader
	err = decoder.Decode(&b)
	if err != nil {
		b.Error = err
		return &b
	}
	for _, a := range b.Authors {
		if StringIsUpper(a.ToString()) {
			b.Error = fmt.Errorf("Author name %s is all in title", a.ToString())
		}
	}
	if StringIsUpper(b.Title) {
		b.Error = fmt.Errorf("Title \"%s\" is all in title", b.Title)
	}
	// for _, author := range v.Authors {
	// 	if len(alist[fingerprint(author)]) == 0 {
	// 		alist[fingerprint(author)] = make(map[Author]struct{})
	// 	}
	// 	alist[fingerprint(author)][author] = struct{}{}
	// }

	return &b
}
