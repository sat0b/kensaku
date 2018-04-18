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

type PostingList map[int][]string

type InvertedIndex map[string][]int

func makeNgram(word string, n int) []string {
	words := make([]string, 0, len(word)/2)
	runes := []rune(word)
	for i := 0; i < len(runes); i += n {
		words = append(words, string(runes[i:i+n]))
	}
	return words
}

func makePostingList(documents []Document) PostingList {
	postingList := make(PostingList)
	for _, document := range documents {
		postingList[document.Id] = makeNgram(document.Title+document.Text, 2)
	}
	return postingList
}

func makeInvertedIndex(postingList PostingList) InvertedIndex {
	invertedIndex := make(InvertedIndex)
	for id, words := range postingList {
		for _, word := range words {
			invertedIndex[word] = append(invertedIndex[word], id)
		}
	}
	return invertedIndex
}

func saveIndex(fileName string) {
	mediaWiki := loadXml(fileName)
	documents := convertDocument(mediaWiki)

	postingList := makePostingList(documents)
	invertedIndex := makeInvertedIndex(postingList)

	b, err := json.Marshal(invertedIndex)
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Create(`/tmp/index.json`)
	defer file.Close()
	file.Write(b)
}

func readIndex(fileName string) InvertedIndex {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal("error: %v", err)
	}

	invertedIndex := make(InvertedIndex, 0)
	json.Unmarshal(data, &invertedIndex)
	return invertedIndex
}

func main() {
	if len(os.Args) != 2 {
		showUsage()
		return
	}
	fileName := os.Args[1]

	saveIndex(fileName)
	invertedIndex := readIndex("/tmp/index.json")
	fmt.Println(invertedIndex)

}
