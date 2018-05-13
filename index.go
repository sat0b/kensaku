package main

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

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
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	decoder := xml.NewDecoder(file)
	pageSave(decoder)
}

func pageSave(decoder *xml.Decoder) {
	const maxPageSize = 100
	pages := make([]Page, 0)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "page" {
				var page Page
				decoder.DecodeElement(&page, &se)
				pages = append(pages, page)
			}
		}

		if len(pages) > maxPageSize {
			documents := convertDocument(pages)
			postingList := makePostingList(documents)
			dictionary := makeDictionary(documents)
			invertedIndex := makeInvertedIndex(postingList)
			saveIndex(invertedIndex)
			saveDictionary(dictionary)
			pages = make([]Page, 0)
		}
	}
}

func convertDocument(pages []Page) []Document {
	documents := make([]Document, 0)
	for i, page := range pages {
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

type OffsetId struct {
	Id     int
	Offset int
}

type InvertedIndex map[string][]OffsetId

type PostingList map[int][]OffsetDoc

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
			invertedIndex[word] = append(invertedIndex[word], OffsetId{Id: id, Offset: offset})
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
	for k, offsetIds := range invertedIndex {
		strv := make([]string, 0)
		for _, offsetId := range offsetIds {
			strId := strconv.Itoa(offsetId.Id)
			strOffset := strconv.Itoa(offsetId.Offset)
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
