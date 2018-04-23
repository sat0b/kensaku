package main

import (
	"log"
	"strconv"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
)

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

func (s *Searcher) Close() {
	s.db.Close()
}

func (s *Searcher) Search(query string) []int {
	result := make([]int, 0)
	resultCount := map[int]int{}
	words := Tokenize(query)
	for _, word := range words {
		ids := s.getResult(word)
		for _, id := range ids {
			if resultCount[id] == 0 {
				result = append(result, id)
				resultCount[id]++
			}
		}
	}
	return result
}

func (s *Searcher) getResult(query string) []int {
	result := make([]int, 0)
	data, err := s.db.Get([]byte(query), nil)
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
