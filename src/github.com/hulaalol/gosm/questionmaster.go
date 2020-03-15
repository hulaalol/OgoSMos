package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/valyala/fastjson"
)

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

func queryDBP(item string) *fastjson.Value {

	Url, err := url.Parse("https://dbpedia.org")
	if err != nil {
		panic("boom")
	}

	Url.Path += "/sparql"
	Url.RawQuery = "default-graph-uri=http://dbpedia.org&query=select+distinct+?property+?value%7B%0D%0A++dbr%3A" + item + "+%3Fproperty+%3Fvalue%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"

	fmt.Printf("Encoded URL is %q\n", Url.String())

	fmt.Println(Url.String())
	var result = getJSON(Url.String())

	//fmt.Println(result)

	return result

}

func getJSON(url string) *fastjson.Value {

	resp, err := http.Get(url)

	if err != nil {
		// handle error
		fmt.Println("error http request")
	}
	body, err := ioutil.ReadAll(resp.Body)

	t := string(body)
	fmt.Println(t)

	var p fastjson.Parser
	v, err := p.Parse(t)

	return v
}

func testGet(url string) string {
	resp, err := http.Get(url)

	if err != nil {
		// handle error
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return string(body)
}
