package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func showUsage() {
	fmt.Println("Usage: kensaku [xmlfile]")
}

type MediaWiki struct {
	Page []Page `xml:"page"`
}

type Page struct {
	Title    string   `xml:"title"`
	Ns       int      `xml:"ns"`
	Revision Revision `xml:"revision"`
}

type Revision struct {
	Text string `xml:"text"`
}

func loadXml(fileName string) *MediaWiki {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal("error: %v", err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal("error: %v", err)
	}

	mediaWiki := new(MediaWiki)
	err = xml.Unmarshal(data, mediaWiki)
	if err != nil {
		log.Fatal(err)
	}

	return mediaWiki
}

func convertDocument(mediaWiki *MediaWiki) []Document {
	var documents []Document
	for i, page := range mediaWiki.Page {
		var document Document
		document.Title = page.Title
		document.Text = page.Revision.Text
		document.Id = i
		documents = append(documents, document)
	}
	return documents
}

type Document struct {
	Title string
	Text  string
	Id    int
}

func saveIndex(fileName string) {
	mediaWiki := loadXml(fileName)
	documents := convertDocument(mediaWiki)

	b, err := json.Marshal(documents)
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Create(`/tmp/index.json`)
	defer file.Close()
	file.Write(b)
}

func readIndex(fileName string) []Document {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal("error: %v", err)
	}

	var documents []Document
	json.Unmarshal(data, &documents)
	return documents
}

func main() {
	if len(os.Args) != 2 {
		showUsage()
		return
	}
	fileName := os.Args[1]

	saveIndex(fileName)
	documents := readIndex("/tmp/index.json")

	for i, document := range documents {
		fmt.Println(i, document)
	}
}
