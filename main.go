package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
)

func showUsage() {
	fmt.Println("Usage: kensaku [xmlfile]")
}

type Response struct {
	Hit       int
	Query     string
	Documents []Document
}

func convertResponseToJson(response Response) string {
	b, err := json.Marshal(response)
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
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
		FeedXml(*filename)
		return
	}

	if *serverMode {
		serve()
	}
}
