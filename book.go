package main

import (
	"archive/zip"
	"crypto/md5"
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
	Error              error
	Path               string
	MD5                string
	Ext                string
	XMLName            xml.Name   `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 FictionBook"`
	ID                 string     `xml:"description>document-info>id"`
	Title              string     `xml:"description>title-info>book-title"`
	Authors            []Person   `xml:"description>title-info>author"`
	Genres             []string   `xml:"description>title-info>genre"`
	AuthorSequences    []Sequence `xml:"description>title-info>sequence"`
	PublisherSequences []Sequence `xml:"description>publish-info>sequence"`
	SrcLang            string     `xml:"description>title-info>src-lang"`
	Keywords           []string   `xml:"description>title-info>keywords"`
	Date               string     `xml:"description>title-info>date"`
	Coverpage          string     `xml:"description>title-info>coverpage"`
	//	 Annotation  string     `xml:"description>title-info>annotation"`
	CorrectAuthors         []Person
	CorrectGenres          []string
	CorrectTitle           string
	CorrectAuthorSequences []Sequence
	Annotation             struct {
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

func (b *Book) Translated() string {
	if strings.ToLower(b.SrcLang) == "ru" || b.SrcLang == "" {
		return "rus"
	} else {
		return "for"
	}
}

type Sequence struct {
	Name   string `xml:"name,attr"`
	Number int    `xml:"number,attr"`
}

func (p *Person) ToString() string {
	return p.Lname + p.Fname
}

func cleanup(s string, title string) string {
	// Remove leading and trailing spaces
	s = strings.TrimSpace(s)
	// Replace unwanted symbols with corret ones
	r := strings.NewReplacer(
		"  ", " ",
		"ё", "е",
		"Ё", "Е",
		"»", "\"",
		"«", "\"")
	s = r.Replace(s)
	if title == "title" {
		s = strings.Title(s)
	}
	return s
	// !"'()+,-.:;=[\]{}Ёё–—
	// words := strings.Fields(s)
	// smallwords := "в на или не х"

	// for index, word := range words {
	// 	if strings.Contains(smallwords, " "+word+" ") {
	// 		words[index] = word
	// } else {
	// 		words[index] = strings.Title(word)
	// 	}
	// }
	// return strings.Join(words, " ")

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
					fmt.Println("b.Error opening file:", b.Error)
					return &b
				}
				break
			}
		}
	case ".fb2":
		xmlFile, b.Error = os.Open(filepath)
		if b.Error != nil {
			fmt.Println("b.Error opening file:", b.Error)
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
	// Add defer to correctly close opened files
	decoder := xml.NewDecoder(xmlFile)
	decoder.CharsetReader = charset.NewReader
	err := decoder.Decode(&b)
	// defer file.Close()

	// result := []byte{}

	hash := md5.New()

	// //Copy the file in the hash interface and check for any error
	_, err = io.Copy(hash, xmlFile)

	if err != nil {
		b.Error = errors.New("Cant read file")
		return &b
	}

	// buf, _ := ioutil.ReadAll(xmlFile)
	// hash.Write(xmlFile)
	fmt.Printf("\n!!! %x, %v !!!\n", hash.Sum(nil))

	// b.MD5 = fmt.Sprintf("%x", hash.Sum(result))

	return cleanUpBook(&b)
}

func cleanUpBook(b *Book) *Book {
	for _, a := range b.Authors {
		newAuthor := Person{}
		newAuthor.Fname = cleanup(a.Fname, "title")
		newAuthor.Lname = cleanup(a.Lname, "title")
		newAuthor.Mname = cleanup(a.Mname, "title")
		b.CorrectAuthors = append(b.CorrectAuthors, newAuthor)
	}

	if len(b.Genres) == 0 {
		b.Genres = append(b.Genres, "Unknown")
	}
	for i := len(b.AuthorSequences) - 1; i >= 0; i-- {
		s := b.AuthorSequences[i]
		// Remove empty.AuthorSequences
		if s.Name == "" {
			b.AuthorSequences = append(b.AuthorSequences[:i], b.AuthorSequences[i+1:]...)
		}
		s.Name = cleanup(s.Name, "")
	}
	if StringIsUpper(b.Title) {
		b.Error = fmt.Errorf("Title \"%s\" is all in title", b.Title)
	}

	if len(b.Authors) > *authcompilation {
		b.CorrectAuthors = []Person{Person{Lname: "Сборник"}}
	}

	return b
}
