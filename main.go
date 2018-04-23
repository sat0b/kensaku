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
		log.Fatal(err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
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

type OffsetIndex struct {
	Id     int
	Offset int
}

type InvertedIndex map[string][]OffsetIndex

type OffsetDoc struct {
	Text   string
	Offset int
}

type PostingList map[int][]OffsetDoc

func Tokenize(word string) []OffsetDoc {
	words := make([]OffsetDoc, 0)
	t := tokenizer.New()
	tokens := t.Tokenize(word)
	for i, token := range tokens {
		if token.Class == tokenizer.DUMMY {
			continue
		}
		words = append(words, OffsetDoc{Text: token.Surface, Offset: i})
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
	for id, offDocs := range postingList {
		for _, offDoc := range offDocs {
			word := offDoc.Text
			offset := offDoc.Offset
			invertedIndex[word] = append(invertedIndex[word], OffsetIndex{Id: id, Offset: offset})
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
	for k, offsetIndexies := range invertedIndex {
		strv := make([]string, 0)
		for _, offsetIndex := range offsetIndexies {
			strId := strconv.Itoa(offsetIndex.Id)
			strOffset := strconv.Itoa(offsetIndex.Offset)
			strv = append(strv, strId+":"+strOffset)
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

func readDictionary(filename string) Dictionary {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
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
	defer searcher.Close()
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
