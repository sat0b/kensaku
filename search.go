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
	offsetDocs := Tokenize(query)
	for _, offsetDoc := range offsetDocs {
		OffsetIds := s.getResult(offsetDoc.Text)
		for _, OffsetId := range OffsetIds {
			if resultCount[OffsetId.Id] == 0 {
				result = append(result, OffsetId.Id)
				resultCount[OffsetId.Id]++
			}
		}
	}
	return result
}

func (s *Searcher) getResult(query string) []OffsetId {
	result := make([]OffsetId, 0)
	data, err := s.db.Get([]byte(query), nil)
	if err == leveldb.ErrNotFound {
		return result
	} else if err != nil {
		log.Fatal(err)
	}
	for _, str := range strings.Split(string(data), ",") {
		splitted := strings.Split(str, ":")
		strId := splitted[0]
		strOffset := splitted[1]
		id, err := strconv.Atoi(strId)
		if err != nil {
			log.Fatal(err)
		}
		offset, err := strconv.Atoi(strOffset)
		if err != nil {
			log.Fatal(err)
		}
		result = append(result, OffsetId{Id: id, Offset: offset})
	}
	return result
}
