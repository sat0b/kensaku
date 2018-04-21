package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/ikawaha/kagome/tokenizer"
)

const indexFilePath = "/tmp/index.json"
const dictionaryFilePath = "/tmp/dictionary.json"

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

type Dictionary map[int]Document

type PostingList map[int][]string

type InvertedIndex map[string][]int

func Tokenize(word string) []string {
	words := make([]string, 0)
	t := tokenizer.New()
	tokens := t.Tokenize(word)
	for _, token := range tokens {
		if token.Class == tokenizer.DUMMY {
			continue
		}
		words = append(words, token.Surface)
	}
	return words
}

func makeDictionary(documents []Document) Dictionary {
	dictionary := make(Dictionary)
	for _, document := range documents {
		dictionary[document.Id] = document
	}
	return dictionary
}

func makePostingList(documents []Document) PostingList {
	postingList := make(PostingList)
	for _, document := range documents {
		postingList[document.Id] = Tokenize(document.Title + document.Text)
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

func feedDocument(fileName string) {
	mediaWiki := loadXml(fileName)
	documents := convertDocument(mediaWiki)

	postingList := makePostingList(documents)
	dictionary := makeDictionary(documents)
	invertedIndex := makeInvertedIndex(postingList)
	saveIndex(invertedIndex)
	saveDictionary(dictionary)
}

func saveIndex(invertedIndex InvertedIndex) {
	b, err := json.Marshal(invertedIndex)
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Create(indexFilePath)
	defer file.Close()
	file.Write(b)
}

func saveDictionary(dictionary Dictionary) {
	b, err := json.Marshal(dictionary)
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Create(dictionaryFilePath)
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

func readDictionary(fileName string) Dictionary {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal("error: %v", err)
	}

	dictionary := make(Dictionary, 0)
	json.Unmarshal(data, &dictionary)
	return dictionary
}

func contain(vec []int, value int) bool {
	for _, v := range vec {
		if v == value {
			return true
		}
	}
	return false
}

func search(invertedIndex InvertedIndex, query string) []int {
	words := Tokenize(query)
	result := make([]int, 0)
	for _, word := range words {
		if ids, ok := invertedIndex[word]; ok {
			for _, id := range ids {
				if !contain(result, id) {
					result = append(result, id)
				}
			}
		}
	}
	return result
}

func printDocument(dictionary Dictionary, documentIds []int) {
	for _, id := range documentIds {
		document := dictionary[id]
		fmt.Printf("Id: %d\n", document.Id)
		fmt.Printf("Title: %s\n", document.Title)
		fmt.Printf("Text: %s\n", document.Text)
	}
}

func main() {
	if len(os.Args) != 2 {
		showUsage()
		return
	}
	fileName := os.Args[1]

	feedDocument(fileName)
	invertedIndex := readIndex(indexFilePath)
	dictionary := readDictionary(dictionaryFilePath)

	for {
		var query string
		fmt.Printf("query: ")
		fmt.Scanf("%s", &query)
		documentIds := search(invertedIndex, query)
		printDocument(dictionary, documentIds)
	}
}
