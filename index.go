package main

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ikawaha/kagome/tokenizer"
	"github.com/syndtr/goleveldb/leveldb"
)

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

func generateIndex(fileName string) {
	mediaWiki := loadXml(fileName)
	documents := convertDocument(mediaWiki)
	postingList := makePostingList(documents)
	dictionary := makeDictionary(documents)
	invertedIndex := makeInvertedIndex(postingList)
	saveIndex(invertedIndex)
	saveDictionary(dictionary)
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
