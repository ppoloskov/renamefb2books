package main

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mgutz/ansi"
	"github.com/paulrosania/go-charset/charset"
	_ "github.com/paulrosania/go-charset/data"
)

type Book struct {
	Error                     error
	Path                      string
	MD5                       string
	Ext                       string
	XMLName                   xml.Name `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 FictionBook"`
	ID                        int
	Title                     string     `xml:"description>title-info>book-title"`
	Authors                   []Person   `xml:"description>title-info>author"`
	Genres                    []string   `xml:"description>title-info>genre"`
	AuthorSequences           []Sequence `xml:"description>title-info>sequence"`
	PublisherSequences        []Sequence `xml:"description>publish-info>sequence"`
	SrcLang                   string     `xml:"description>title-info>src-lang"`
	Keywords                  []string   `xml:"description>title-info>keywords"`
	Date                      string     `xml:"description>title-info>date"`
	Coverpage                 string     `xml:"description>title-info>coverpage"`
	Lang                      string     `xml:"description>title-info>lang"`
	Translators               []Person   `xml:"description>title-info>translator"`
	CorrectAuthors            []Person
	CorrectGenres             []string
	CorrectTitle              string
	CorrectAuthorSequences    []Sequence
	CorrectPublisherSequences []Sequence
	Translated                bool

	// ID                 string     `xml:"description>document-info>id"`
	//	 Annotation  string     `xml:"description>title-info>annotation"`
	// Annotation             struct {
	// 	Items []struct {
	// 		XMLName xml.Name
	// 		Content string `xml:",innerxml"`
	// 	} `xml:",any"`
	// } `xml:"description>title-info>annotation"`
	// rest        []string `xml:,any`
}

func (b Book) String() string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("Book path: %s, extention: %s, MD5: %d\n", b.Path, b.Ext, b.ID))

	buffer.WriteString(ansi.ColorCode("green+h:black") + "Book original/corrected title: " + ansi.ColorCode("reset"))
	buffer.WriteString(b.Title)
	buffer.WriteString(ansi.ColorCode("green+h:black") + " / " + ansi.ColorCode("reset"))
	buffer.WriteString(b.CorrectTitle)
	buffer.WriteString("\n")

	buffer.WriteString(ansi.ColorCode("green+h:black") + "Original/corrected authors: " + ansi.ColorCode("reset"))
	for i, a := range b.Authors {
		buffer.WriteString(fmt.Sprintf("%s", a))
		if i < len(b.Authors)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString(ansi.ColorCode("green+h:black") + " / " + ansi.ColorCode("reset"))
	for i, a := range b.CorrectAuthors {
		buffer.WriteString(fmt.Sprintf("%s", a))
		if i < len(b.CorrectAuthors)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString("\n")

	buffer.WriteString(ansi.ColorCode("green+h:black") + "Original/corrected Genres: " + ansi.ColorCode("reset"))
	for i, g := range b.Genres {
		buffer.WriteString(fmt.Sprintf("%s", g))
		if i < len(b.Genres)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString(ansi.ColorCode("green+h:black") + " / " + ansi.ColorCode("reset"))
	for i, g := range b.CorrectGenres {
		buffer.WriteString(fmt.Sprintf("%s", g))
		if i < len(b.CorrectGenres)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString("\n")

	buffer.WriteString(ansi.ColorCode("green+h:black") + "Original/corrected author sequences: " + ansi.ColorCode("reset"))
	for i, s := range b.AuthorSequences {
		buffer.WriteString(fmt.Sprintf("%s", s))
		if i < len(b.AuthorSequences)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString(ansi.ColorCode("green+h:black") + " / " + ansi.ColorCode("reset"))
	for i, s := range b.CorrectAuthorSequences {
		buffer.WriteString(fmt.Sprintf("%s", s))
		if i < len(b.CorrectAuthorSequences)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString("\n")

	buffer.WriteString(ansi.ColorCode("green+h:black") + "Original/corrected publisher sequences: " + ansi.ColorCode("reset"))
	for i, s := range b.PublisherSequences {
		buffer.WriteString(fmt.Sprintf("%s", s))
		if i < len(b.PublisherSequences)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString(ansi.ColorCode("green+h:black") + " / " + ansi.ColorCode("reset"))
	for i, s := range b.CorrectPublisherSequences {
		buffer.WriteString(fmt.Sprintf("%s", s))
		if i < len(b.CorrectPublisherSequences)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString("\n")

	// SrcLang                   string     `xml:"description>title-info>src-lang"`
	// Date                      string     `xml:"description>title-info>date"`
	// Lang                      string     `xml:"description>title-info>lang"`
	// Translators               []Person   `xml:"description>title-info>translator"`

	if b.Translated {
		buffer.WriteString("Translated\n")
	} else {
		buffer.WriteString("Not translated\n")
	}
	return buffer.String()
}

// // <description> - который описывает заголовок документа. Одно и только одно вхождение. (фразы вроде "одно и только одно вхождение" говорят, сколько раз подряд может идти данный тэг в данном месте документа)
// <body> - описывает тело документа. Одно или более вхождений.
// <binary>

// <src-title-info> - данные об исходнике книги (до перевода). От нуля до одного вхождений.
// <document-info> - информация об FB2-документе. Одно и только одно вхождение.
// <publish-info> - сведения об издании книги, которая была использована как источник при подготовке документа. От нуля до одного вхождений.
// <custom-info>

// func (b *Book) Translated() string {
// 	if strings.ToLower(b.SrcLang) == "ru" || b.SrcLang == "" {
// 		return "rus"
// 	} else {
// 		return "for"
// 	}
// }

type Sequence struct {
	LRSId       int
	Name        string `xml:"name,attr"`
	Number      int    `xml:"number,attr"`
	BookCounter int
}

func (s Sequence) String() string {
	return fmt.Sprintf("ID: %d, %s (#%d), Books no. %d", s.LRSId, s.Name, s.Number, s.BookCounter)
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

		for _, f := range r.File {
			if path.Ext(f.Name) == ".fb2" {
				xmlFile, b.Error = f.Open()
				// defer f.Close()
				break
			}
		}

	case ".fb2":
		xmlFile, b.Error = os.Open(filepath)

	default:
		b.Error = errors.New("Wrong file")
		return &b
	}

	if b.Error != nil {
		fmt.Println("b.Error opening file:", b.Error)
		return &b
	}

	if xmlFile == nil {
		b.Error = errors.New("Wrong file")
		return &b
	}

	buf, err := ioutil.ReadAll(xmlFile)
	x := bytes.NewBuffer(buf)
	// Add defer to correctly close opened files

	decoder := xml.NewDecoder(x)
	decoder.CharsetReader = charset.NewReader
	err = decoder.Decode(&b)

	// defer file.Close()

	x = bytes.NewBuffer(buf)
	b.MD5 = fmt.Sprintf("%x", md5.Sum(x.Bytes()))

	if err != nil {
		b.Error = errors.New("Cant read file")
		return &b
	}

	// return cleanUpBook(&b)
	return &b
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
