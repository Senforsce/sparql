// Package sparql contains functions and data structures for querying SPARQL
// endpoints and parsing the response into RDF terms, as well as other
// convenience functions for working with SPARQL queries.
package sparql

import (
	"strings"
	"time"

	"github.com/knakk/rdf"
)

// DateFormat is the expected layout of the xsd:DateTime values. You can override
// it if your triple store uses a different layout.
var DateFormat = time.RFC3339

var xsdString rdf.IRI

func init() {
	xsdString, _ = rdf.NewIRI("http://www.w3.org/2001/XMLSchema#string")
}

func FindObjectValueByPredicate(needle string, haystack []map[string]Binding) map[string]Binding {
	for _, term := range haystack {
		// Check if the predicate (p) matches the needle
		if val, exists := term["p"]; exists {
			if strings.Contains(val.Value, needle) {
				// If a match is found, return the object (o) and true
				return term
			}
		}
	}
	// If no match is found, return an empty map and false
	return map[string]Binding{}
}

func GetValue(needle string, haystack []map[string]Binding) string {
	if len(haystack) == 0 {
		return ""
	}

	term := haystack[0]

	// Check if the predicate (p) matches the needle
	if val, exists := term[needle]; exists {
		return val.Value
	}

	// If no match is found, return an empty map and false
	return ""
}

func FindObjectValueBySpecifiedPredicate(needle string, predicate string, haystack []map[string]Binding) map[string]Binding {
	for _, term := range haystack {
		// Check if the predicate (p) matches the needle
		if val, exists := term[predicate]; exists {
			if strings.Contains(val.Value, needle) {
				// If a match is found, return the object (o) and true
				return term
			}
		}
	}
	// If no match is found, return an empty map and false
	return map[string]Binding{}
}

func ListOfSubjects(results []map[string]Binding) map[string][]map[string]Binding {
	var toReturn = make(map[string][]map[string]Binding)

	for _, term := range results {
		// Check if the predicate (p) matches the needle
		if _, ok := term["s"]; ok {
			val := term["s"].Value

			if len(toReturn[val]) < 1 {
				toReturn[val] = []map[string]Binding{}
			}
			toReturn[val] = append(toReturn[val], term)
		}
	}

	return toReturn
}

func ListOf(results []map[string]Binding, needle string) map[string][]map[string]Binding {
	var toReturn = make(map[string][]map[string]Binding)

	for _, term := range results {
		// Check if the predicate (p) matches the needle
		if _, ok := term[needle]; ok {
			val := term[needle].Value

			if len(toReturn[val]) < 1 {
				toReturn[val] = []map[string]Binding{}
			}
			toReturn[val] = append(toReturn[val], term)
		}
	}

	return toReturn
}
