package sparql

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/knakk/digest"
	"github.com/knakk/rdf"
)

var ct = "Content-Type"
var cl = "Content-Length"
var rb = "Response body: \n"
var readFailMsg = "Failed to read response body"
var ctvalue = "application/x-www-form-urlencoded"

// Repo represent a RDF repository, assumed to be
// queryable via the SPARQL protocol over HTTP.
type Repo struct {
	endpoint string
	client   *http.Client
}

type header struct {
	Link []string
	Vars []string
}

type Results struct {
	Head    header
	Results results
}

type Binding struct {
	Type     string // "uri", "literal", "typed-literal" or "bnode"
	Value    string
	Lang     string `json:"xml:lang"`
	DataType string
}

type results struct {
	Distinct bool
	Ordered  bool
	Bindings []map[string]Binding
}

// ParseJSON takes an application/sparql-results+json response and parses it
// into a Results struct.
func ParseJSON(r io.Reader) (*Results, error) {
	var res Results
	err := json.NewDecoder(r).Decode(&res)

	return &res, err
}

// Bindings returns a map of the bound variables in the SPARQL response, where
// each variable points to one or more RDF terms.
func (r *Results) Bindings() map[string][]rdf.Term {
	rb := make(map[string][]rdf.Term)
	for _, v := range r.Head.Vars {
		for _, b := range r.Results.Bindings {
			t, err := termFromJSON(b[v])
			if err == nil {
				rb[v] = append(rb[v], t)
			}
		}
	}

	return rb
}

// Solutions returns a slice of the query solutions, each containing a map
// of all bindings to RDF terms.
func (r *Results) Solutions() []map[string]rdf.Term {
	var rs []map[string]rdf.Term

	for _, s := range r.Results.Bindings {
		solution := make(map[string]rdf.Term)
		for k, v := range s {
			term, err := termFromJSON(v)
			if err == nil {
				solution[k] = term
			}
		}
		rs = append(rs, solution)
	}

	return rs
}

// termFromJSON converts a SPARQL json result binding into a rdf.Term. Any
// parsing errors on typed-literal will result in a xsd:string-typed RDF term.
// TODO move this functionality to package rdf?
func termFromJSON(b Binding) (rdf.Term, error) {
	switch b.Type {
	case "bnode":
		return rdf.NewBlank(b.Value)
	case "uri":
		return rdf.NewIRI(b.Value)
	case "literal":
		// Untyped literals are typed as xsd:string
		if b.Lang != "" {
			return rdf.NewLangLiteral(b.Value, b.Lang)
		}
		return rdf.NewTypedLiteral(b.Value, xsdString), nil
	case "typed-literal":
		iri, err := rdf.NewIRI(b.DataType)
		if err != nil {
			return nil, err
		}
		return rdf.NewTypedLiteral(b.Value, iri), nil
	default:
		return nil, errors.New("unknown term type")
	}
}

// NewRepo creates a new representation of a RDF repository. It takes a
// variadic list of functional options which can alter the configuration
// of the repository.
func NewRepo(addr string, options ...func(*Repo) error) (*Repo, error) {
	r := Repo{
		endpoint: addr,
		client:   http.DefaultClient,
	}
	return &r, r.SetOption(options...)
}

// SetOption takes one or more option function and applies them in order to Repo.
func (r *Repo) SetOption(options ...func(*Repo) error) error {
	for _, opt := range options {
		if err := opt(r); err != nil {
			return err
		}
	}
	return nil
}

// DigestAuth configures Repo to use digest authentication on HTTP requests.
func DigestAuth(username, password string) func(*Repo) error {
	return func(r *Repo) error {
		r.client.Transport = digest.NewTransport(username, password)
		return nil
	}
}

// Timeout instructs the underlying HTTP transport to timeout after given duration.
func Timeout(t time.Duration) func(*Repo) error {
	return func(r *Repo) error {
		r.client.Timeout = t
		return nil
	}
}

// Query performs a SPARQL HTTP request to the Repo, and returns the
// parsed application/sparql-results+json response.
func (r *Repo) Query(q string) (*Results, error) {
	form := url.Values{}
	form.Set("query", q)
	b := form.Encode()

	// TODO make optional GET or Post, Query() should default GET (idempotent, cacheable)
	// maybe new for updates: func (r *Repo) Update(q string) using POST?
	req, err := http.NewRequest(
		"POST",
		r.endpoint,
		bytes.NewBufferString(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set(ct, ctvalue)
	req.Header.Set(cl, strconv.Itoa(len(b)))
	req.Header.Set("Accept", "application/sparql-results+json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(resp.Body)
		var msg string
		if err != nil {
			msg = readFailMsg
		} else {
			if strings.TrimSpace(string(b)) != "" {
				msg = rb + string(b)
			}
		}
		return nil, fmt.Errorf("Query: SPARQL request failed: %s. "+msg, resp.Status)
	}
	results, err := ParseJSON(resp.Body)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *Repo) Update(q string) (string, error) {
	form := url.Values{}
	form.Set("update", q)
	b := form.Encode()

	// TODO make optional GET or Post, Query() should default GET (idempotent, cacheable)
	// maybe new for updates: func (r *Repo) Update(q string) using POST?
	req, err := http.NewRequest(
		"POST",
		r.endpoint,
		bytes.NewBufferString(b))
	if err != nil {
		return "", err
	}

	req.Header.Set(ct, ctvalue)
	req.Header.Set(cl, strconv.Itoa(len(b)))
	req.Header.Set("Accept", "application/sparql-results+json")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(resp.Body)
		var msg string
		if err != nil {
			msg = readFailMsg
		} else {
			if strings.TrimSpace(string(b)) != "" {
				msg = rb + string(b)
			}
		}
		return "", fmt.Errorf("Query: SPARQL request failed: %s. "+msg, resp.Status)
	}
	result := "OK"

	return result, nil
}

// Construct performs a SPARQL HTTP request to the Repo, and returns the
// result triples.
func (r *Repo) Construct(q string) ([]rdf.Triple, error) {
	form := url.Values{}
	form.Set("query", q)
	form.Set("format", "text/turtle")
	b := form.Encode()

	req, err := http.NewRequest(
		"POST",
		r.endpoint,
		bytes.NewBufferString(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set(ct, ctvalue)
	req.Header.Set(cl, strconv.Itoa(len(b)))
	req.Header.Set("Accept", "text/turtle")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(resp.Body)
		var msg string
		if err != nil {
			msg = readFailMsg
		} else {
			if strings.TrimSpace(string(b)) != "" {
				msg = rb + string(b)
			}
		}
		return nil, fmt.Errorf("Construct: SPARQL request failed: %s. "+msg, resp.Status)
	}
	dec := rdf.NewTripleDecoder(resp.Body, rdf.Turtle)
	return dec.DecodeAll()
}
