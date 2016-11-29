package main

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/analysis/datetime/flexible"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/bleve/registry"
)

const dateTimeParserName = "dateTimeParser"

func init() {
	registry.RegisterDateTimeParser(dateTimeParserName, DateTimeParserConstructor)
}

var dateTimeParserLayouts = []string{
	time.RFC1123Z,
}

func DateTimeParserConstructor(config map[string]interface{}, cache *registry.Cache) (analysis.DateTimeParser, error) {
	return flexible.New(dateTimeParserLayouts), nil
}

func buildIndexMapping() mapping.IndexMapping {
	mapping := bleve.NewIndexMapping()

	dateFieldMapping := bleve.NewDateTimeFieldMapping()
	dateFieldMapping.DateFormat = dateTimeParserName

	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name

	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("Type", keywordFieldMapping)

	headerMapping := bleve.NewDocumentMapping()
	headerMapping.AddFieldMappingsAt("Subject", keywordFieldMapping)
	headerMapping.AddFieldMappingsAt("From", keywordFieldMapping)
	headerMapping.AddFieldMappingsAt("To", keywordFieldMapping)
	headerMapping.AddFieldMappingsAt("Date", dateFieldMapping)

	docMapping.AddSubDocumentMapping("Data", headerMapping)

	mapping.AddDocumentMapping("header", docMapping)
	mapping.TypeField = "Type"

	return mapping
}
