package main

import (
	"fmt"
	"time"

	"github.com/blevesearch/bleve/analysis"
	"github.com/blevesearch/bleve/analysis/datetime/flexible"
	"github.com/blevesearch/bleve/registry"
)

const dateTimeParserName = "dateTimeParser"

var dateTimeParserLayouts = []string{
	time.RFC1123Z,
}

func DateTimeParserConstructor(config map[string]interface{}, cache *registry.Cache) (analysis.DateTimeParser, error) {
	fmt.Println("time parser called")
	return flexible.New(dateTimeParserLayouts), nil
}

func init() {
	registry.RegisterDateTimeParser(dateTimeParserName, DateTimeParserConstructor)
}
