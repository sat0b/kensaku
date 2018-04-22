package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ikawaha/kagome/tokenizer"
	"github.com/syndtr/goleveldb/leveldb"
)

const indexFilePath = "/tmp/index.db"
const dictionaryFilePath = "/tmp/dictionary.db"

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
	documents := make([]Document, 0)
	for i, page := range mediaWiki.Page {
		document := Document{Title: page.Title, Text: page.Revision.Text, Id: i}
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
	db, err := leveldb.OpenFile(indexFilePath, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	for k, v := range invertedIndex {
		strv := make([]string, 0)
		for _, id := range v {
			strid := strconv.Itoa(id)
			strv = append(strv, strid)
		}
		strids := strings.Join(strv, ",")
		err = db.Put([]byte(k), []byte(strids), nil)
		if err != nil {
			log.Fatal(err)
		}
	}
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

type Searcher struct {
	db *leveldb.DB
}

func NewSearcher(filename string) *Searcher {
	searcher := new(Searcher)
	db, err := leveldb.OpenFile(filename, nil)
	if err != nil {
		log.Fatal(err)
	}
	searcher.db = db
	return searcher
}

func (searcher *Searcher) Close() {
	searcher.db.Close()
}

func (searcher *Searcher) Search(query string) []int {
	result := make([]int, 0)
	resultCount := map[int]int{}
	words := Tokenize(query)
	for _, word := range words {
		ids := searcher.getResult(word)
		for _, id := range ids {
			if resultCount[id] == 0 {
				result = append(result, id)
				resultCount[id]++
			}
		}
	}
	return result
}

func (searcher *Searcher) getResult(query string) []int {
	result := make([]int, 0)
	data, err := searcher.db.Get([]byte(query), nil)
	if err == leveldb.ErrNotFound {
		return result
	} else if err != nil {
		log.Fatal(err)
	}
	for _, strid := range strings.Split(string(data), ",") {
		id, err := strconv.Atoi(strid)
		if err != nil {
			log.Fatal(err)
		}
		result = append(result, id)
	}
	return result
}

func readDictionary(filename string) Dictionary {
	file, err := os.Open(filename)
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

func convertResponseToJson(response Response) string {
	b, err := json.Marshal(response)
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}

type Response struct {
	Hit       int
	Query     string
	Documents []Document
}

func getJsonOutput(query string, dictionary Dictionary, documentIds []int) string {
	documents := make([]Document, 0)
	for _, id := range documentIds {
		documents = append(documents, dictionary[id])
	}
	response := Response{Hit: len(documentIds), Query: query, Documents: documents}
	output := convertResponseToJson(response)
	return output
}

func serve() {
	dictionary := readDictionary(dictionaryFilePath)
	searcher := NewSearcher(indexFilePath)
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := r.ParseForm(); err != nil {
			log.Print(err)
		}
		if query, ok := r.Form["query"]; ok {
			q := query[0]
			documentIds := searcher.Search(q)
			output := getJsonOutput(q, dictionary, documentIds)
			fmt.Fprintf(w, output)
		}
	})
	http.ListenAndServe(":8000", nil)
}

func main() {
	filename := flag.String("f", "", "feed data (xml)")
	serverMode := flag.Bool("s", true, "server mode")
	flag.Parse()

	if *filename != "" {
		feedDocument(*filename)
		return
	}

	if *serverMode {
		serve()
	}
}
