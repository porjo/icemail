package main

import (
	"net/mail"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis"
	"github.com/blevesearch/bleve/analysis/datetime/flexible"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/bleve/registry"
)

type bleveDoc struct {
	Type   string
	Header mail.Header
	// store raw email data
	Data      string
	Delivered time.Time
}

// locationsBase is prepended to locations being filtered on
const locationsBase = "Header."

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

	headerMapping := bleve.NewDocumentMapping()
	dateFieldMapping := bleve.NewDateTimeFieldMapping()
	dateFieldMapping.DateFormat = dateTimeParserName
	headerMapping.AddFieldMappingsAt("Date", dateFieldMapping)

	docMapping := bleve.NewDocumentMapping()
	dataFieldMapping := bleve.NewTextFieldMapping()
	dataFieldMapping.Index = false
	docMapping.AddFieldMappingsAt("Data", dataFieldMapping)
	docMapping.AddSubDocumentMapping("Header", headerMapping)

	mapping.AddDocumentMapping("message", docMapping)
	mapping.TypeField = "Type"

	return mapping
}
