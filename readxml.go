package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	// "strings"

	"github.com/beevik/etree"
	"reflect"
)

// func TestSave(filename string) {
// 	doc := New()

// 	if err := doc.LoadFile("test.xml", nil); err != nil {
// 		t.Errorf("LoadFile(): %s", err)
// 		return
// 	}

// 	IndentPrefix = "\t"
// 	if err := doc.SaveFile("test1.xml"); err != nil {
// 		t.Errorf("SaveFile(): %s", err)
// 		return
// 	}
// }

func (aut Person) CreateXml(e *etree.Element) *etree.Element {
	a := e.CreateElement("author")
	a_ln := a.CreateElement("last-name")
	a_mn := a.CreateElement("middle-name")
	a_fn := a.CreateElement("first-name")
	a_ln.SetText(aut.Lname)
	a_mn.SetText(aut.Mname)
	a_fn.SetText(aut.Fname)
	return a
}

func NodeSearch(filename string) {

	doc := etree.NewDocument()

	if err := doc.ReadFromFile(filename); err != nil {
		panic(err)
	}

	aut := Person{Fname: "ПЕТР", Mname: "ПЕТРОВИЧ", Lname: "ПЕТУХОВ"}
	root := doc.FindElement("./FictionBook/description/title-info")

	fmt.Println("ROOT element:", root.Tag)
	aut.CreateXml(root)

	// doc := xmlx.New()
	fo, err := os.Create("output.fb2")
	if err != nil {
		panic(err)
	}
	doc.IndentTabs()
	doc.WriteTo(fo)

	// if err := doc.LoadFile(filename, nil); err != nil {
	// 	fmt.Printf("LoadFile(): %s", err)
	// 	return
	// }
	// node := doc.SelectNode("http://www.gribuser.ru/xml/fictionbook/2.0", "middle-name")
	// if node == nil {
	// 	fmt.Printf("SelectNode(): No node found.")
	// 	return
	// }
	// // addTo := node.Parent

	// fmt.Printf("%v\n", node.Type)
}

func modifyXmlValues(content io.Reader) []byte {
	var buffer bytes.Buffer
	// inputReader := strings.NewReader(content)
	decoder := xml.NewDecoder(content)
	encoder := xml.NewEncoder(&buffer)
	encoder.Indent("", " ")
	// buffer.WriteString(xml.Header)
	// buffer.WriteString("<!DOCTYPE tsung SYSTEM '/usr/share/tsung/tsung-1.0.dtd'>\n")
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		// switch token := t.(type) {
		// case xml.StartElement:
		// 	switch token.Name.Local {
		// 	case "load":
		// 		encoder.Encode(load)
		// 	case "clients":
		// 		encoder.Encode(cloud)
		// 	default:
		// 		err := encoder.EncodeToken(t)
		// 		if err != nil {
		// 			fmt.Printf("error=%s\b", err.Error())
		// 		}
		// 	}
		// case xml.EndElement:
		//t{Name[0]} = ""

		if reflect.TypeOf(t).String() == "xml.StartElement" {
			if t.(xml.StartElement).Name.Local == "author" {
				aut := Person{Fname: "ПЕТР", Mname: "ПЕТРОВИЧ", Lname: "ПЕТУХОВ"}
				err := encoder.Encode(aut)
				if err != nil {
					fmt.Printf("! error=%s\b", err.Error())
				}
			} else {
				err := encoder.EncodeToken(t)
				if err != nil {
					fmt.Printf("error=%s\b", err.Error())
				}
			}
			// decoder.DecodeElement(info, &startElement)
			// &t.(xml.StartElement).Name = xml.Name{Space: "DAV:", Local: "multistatus"}
			// fmt.Printf("%v\n\n", reflect.ValueOf(t.(xml.StartElement).Name))
		} else {
			err := encoder.EncodeToken(t)
			if err != nil {
				fmt.Printf("error=%s\b", err.Error())
			}
		}
	}
	encoder.Flush()
	return buffer.Bytes()
	// return bytes.NewReader(buffer.Bytes())
}

// func main() {
//
// 	flag.Parse()
// 	file := flag.Arg(0)
// 	NodeSearch(file)
//
// 	// xmlFile, err := os.Open(file)
// 	// if err != nil {
// 	// 	fmt.Println("Error opening file:", err)
// 	// 	return
// 	// }
// 	// defer xmlFile.Close()
//
// 	// // decoder := xml.NewDecoder(xmlFile)
//
// 	// bf := modifyXmlValues(xmlFile)
//
// 	// // open output file
// 	// fo, err := os.Create("output.txt")
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	// // close fo on exit and check for its returned error
// 	// defer func() {
// 	// 	if err := fo.Close(); err != nil {
// 	// 		panic(err)
// 	// 	}
// 	// }()
// 	// // write a chunk
// 	// if _, err := fo.Write(bf); err != nil {
// 	// 	panic(err)
// 	// }
//
// }
