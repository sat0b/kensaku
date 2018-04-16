package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

func showUsage() {
	fmt.Println("Usage: kensaku [xmlfile]")
}

func main() {
	if len(os.Args) != 2 {
		showUsage()
		return
	}
	fileName := os.Args[1]

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

	fmt.Printf("%+v\n", mediaWiki)
}
