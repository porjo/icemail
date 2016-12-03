package main

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis"
	"github.com/blevesearch/bleve/analysis/datetime/flexible"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/bleve/registry"
)

const dateTimeParserName = "dateTimeParser"
const RFC1123ZnoPadDay = "Mon, _2 Jan 2006 15:04:05 -0700"

func init() {
	registry.RegisterDateTimeParser(dateTimeParserName, DateTimeParserConstructor)
}

var dateTimeParserLayouts = []string{
	RFC1123ZnoPadDay,
}

func DateTimeParserConstructor(config map[string]interface{}, cache *registry.Cache) (analysis.DateTimeParser, error) {
	return flexible.New(dateTimeParserLayouts), nil
}

func buildIndexMapping() mapping.IndexMapping {
	mapping := bleve.NewIndexMapping()

	docMapping := bleve.NewDocumentMapping()

	headerMapping := bleve.NewDocumentMapping()
	dateFieldMapping := bleve.NewDateTimeFieldMapping()
	dateFieldMapping.DateFormat = dateTimeParserName
	headerMapping.AddFieldMappingsAt("Date", dateFieldMapping)
	dataFieldMapping := bleve.NewTextFieldMapping()
	dataFieldMapping.Index = false
	headerMapping.AddFieldMappingsAt("Data", dataFieldMapping)
	docMapping.AddSubDocumentMapping("Header", headerMapping)

	mapping.AddDocumentMapping("message", docMapping)
	mapping.TypeField = "Type"

	return mapping
}
