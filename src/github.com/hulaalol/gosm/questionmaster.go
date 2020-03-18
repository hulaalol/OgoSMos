package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/valyala/fastjson"
)

type information struct {
	typ info
	val info
}

type info struct {
	typ string
	val string
}

func getJSON(url string) *fastjson.Value {

	resp, err := http.Get(url)
	if err != nil {
		// handle error
		fmt.Println("error http request")
	}
	body, err := ioutil.ReadAll(resp.Body)
	t := string(body)

	var p fastjson.Parser
	v, err := p.Parse(t)
	return v
}

func queryDBP(item string) []information {

	Url, err := url.Parse("https://dbpedia.org")
	if err != nil {
		panic("boom")
	}

	// Raw query because DBpedia query special characters should NOT be html encoded
	Url.Path += "/sparql"
	Url.RawQuery = "default-graph-uri=http://dbpedia.org&query=select+distinct+?property+?value%7B%0D%0A++dbr%3A" + item + "+%3Fproperty+%3Fvalue%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	//fmt.Println("Querying DBPedia with: " + Url.String())

	// create json from json-string answer
	var data = getJSON(Url.String()).Get("results", "bindings").GetArray()

	// convert json structure to go-objects array (information array)
	var res []information
	for _, d := range data {

		var p = info{string(d.Get("property").GetStringBytes("type")), string(d.Get("property").GetStringBytes("value"))}
		var v = info{string(d.Get("value").GetStringBytes("type")), string(d.Get("value").GetStringBytes("value"))}
		res = append(res, information{p, v})
	}

	fmt.Println("Found " + strconv.Itoa(len(res)) + " informations about " + item)
	return res
}

var stopwords = [...]string{"weg", "straße", "strasse", "allee", "gasse", "Straße", "Weg", "Strasse", "Allee"}
var syllables = [...]string{"er"}

func cleanStreetname(s string) string {
	for _, sw := range stopwords {
		s = strings.ReplaceAll(s, sw, "")
	}

	for _, sy := range syllables {

		var syLength = len(sy)
		var sLength = len(s)

		if strings.Compare(s[sLength-1-syLength:sLength], sy) == 1 {
			s = s[0 : sLength-1-syLength]
		}
	}
	return s
}
