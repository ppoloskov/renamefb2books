package main

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/paulrosania/go-charset/charset"
	_ "github.com/paulrosania/go-charset/data"
)

type Book struct {
	Error     error
	Path      string
	Ext       string
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

func parsefb2(filepath string) *Book {
	b := Book{}
	b.Path = filepath
	b.Ext = path.Ext(filepath)
	var xmlFile io.Reader

	switch b.Ext {
	case ".zip":
		var r *zip.ReadCloser
		if r, b.Error = zip.OpenReader(filepath); b.Error != nil {
			fmt.Println("b.Error opening file:", b.Error)
			return &b
		}
		defer r.Close()

		for _, f := range r.File {
			if path.Ext(f.Name) == ".fb2" {
				if xmlFile, b.Error = f.Open(); b.Error != nil {
					fmt.Println("b.Erroror opening file:", b.Error)
					return &b
				}
				break
			}
		}
	case ".fb2":
		xmlFile, b.Error = os.Open(filepath)
		if b.Error != nil {
			fmt.Println("b.Erroror opening file:", b.Error)
			return &b
		}
	default:
		b.Error = errors.New("Wrong file")
		return &b
	}

	if xmlFile == nil {
		b.Error = errors.New("Wrong file")
		return &b
	}

	decoder := xml.NewDecoder(xmlFile)
	decoder.CharsetReader = charset.NewReader
	fmt.Println(filepath)
	err := decoder.Decode(&b)
	if err != nil {
		b.Error = err
		return &b
	}
	for i, a := range b.Authors {
		b.Authors[i].Fname = strings.TrimSpace(b.Authors[i].Fname)
		b.Authors[i].Lname = strings.TrimSpace(b.Authors[i].Lname)
		b.Authors[i].Mname = strings.TrimSpace(b.Authors[i].Mname)
		if StringIsUpper(a.ToString()) {
			b.Error = fmt.Errorf("Author name %s is all in title", a.ToString())
		}
	}
	if StringIsUpper(b.Title) {
		b.Error = fmt.Errorf("Title \"%s\" is all in title", b.Title)
	}

	if len(b.Authors) > *authcompilation {
		b.Authors = []Person{Person{Lname: "Сборник"}}
	}

	return &b
}
