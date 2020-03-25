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
	//fmt.Println(url)
	if err != nil {
		// handle error
		fmt.Println("error http request")
	}
	body, err := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))
	t := string(body)

	var p fastjson.Parser
	v, err := p.Parse(t)
	return v
}

func queryDBP(item string, typ string) []information {

	Url, err := url.Parse("https://dbpedia.org")
	if err != nil {
		panic("boom")
	}

	// Raw query because DBpedia query special characters should NOT be html encoded
	Url.Path += "/sparql"
	Url.RawQuery = "default-graph-uri=http://dbpedia.org&query=select+distinct+?property+?value%7B%0D%0A++" + typ + "%3A" + item + "+%3Fproperty+%3Fvalue%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
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

	var redirect = getRedirect(res)

	if redirect != "null" {
		fmt.Println("Found redirect. Following...")

		res = queryDBP(redirect, typ)
	}

	return res
}

func getRedirect(info []information) string {

	var redirect = filterInfo(info, []string{"wikiPageRedirects"}, []string{})

	if len(redirect) > 0 {
		return cleanURL(redirect[0].val.val)
	} else {
		return "null"
	}

}

func filterInfo(info []information, filterType []string, filterValue []string) []information {
	var res []information
	for _, i := range info {
		for _, f := range filterType {
			if strings.Contains(i.typ.val, f) {

				if len(filterValue) > 0 {
					for _, fV := range filterValue {
						if strings.Contains(i.val.val, fV) {
							res = append(res, i)
						}
					}
				} else {
					res = append(res, i)
				}

			}
		}
	}
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

func cleanURL(s string) string {

	var lastSlash = 0
	for i, _ := range s {
		if s[i] == 47 {
			lastSlash = i
		}
	}
	return s[lastSlash+1 : len(s)]
}

func cleanCategory(s string) string {
	var lastColon = 0
	for i, _ := range s {
		if s[i] == 58 {
			lastColon = i
		}
	}
	return s[lastColon+1 : len(s)]
}

type Question struct {
	item        string
	answer      string
	distractors [3]string
}

type ItemData struct {
	class          information
	superClasses   []information
	siblingClasses []info
	subjects       []information
	properties     []information
}

func getClassName(queryResult []information) []string {
	// determine the proper class of the item
	var classRDF = filterInfo(queryResult, []string{"rdf-syntax-ns#type", "rdf-schema#subClassOf"}, []string{"dbpedia.org/ontology", "owl#Class"})
	var classDBO = filterInfo(queryResult, []string{"dbpedia.org/ontology/type", "rdf-schema#subClassOf"}, []string{"dbpedia.org"})
	var class []information
	if len(classDBO) == 0 && len(classRDF) == 0 {
		fmt.Println("Could not find class!")
	}
	if (len(classRDF) > 0) && (len(classRDF) < len(classDBO) || len(classDBO) == 0) {
		class = classRDF
	} else {
		if len(classDBO) == 0 {
			fmt.Println("Could not find class!")
		} else {
			class = classDBO

		}
	}
	var className = class[0].val.val

	var typ string

	if strings.Contains(className, "ontology") {
		typ = "dbo"
	} else {
		typ = "dbr"
	}

	return []string{cleanURL(className), typ}
}

func genItem(item string) ItemData {

	var res = queryDBP(item, "dbr")
	if len(res) == 0 {
		res = queryDBP(item, "dbo")
	}

	var className = getClassName(res)
	fmt.Println("Found class " + className[0] + " for " + item)

	// find superclasses and siblings

	var test = strings.ReplaceAll(className[0], "(", "\\(")
	test = strings.ReplaceAll(test, ")", "\\)")

	className[0] = test

	var classQuery = queryDBP(className[0], className[1])
	var superClass = getClassName(classQuery)
	fmt.Println("Found superclass " + superClass[0] + " for " + item)

	return ItemData{}
}

func queryDBPSiblings(ontology string, category string) []info {

	Url, err := url.Parse("https://dbpedia.org")
	if err != nil {
		panic("boom")
	}

	// Raw query because DBpedia query special characters should NOT be html encoded
	Url.Path += "/sparql"
	Url.RawQuery = "default-graph-uri=http://dbpedia.org&query=select+?s1%0D%0Awhere+%7B%0D%0A++++%3Fs1+a+dbo%3A" + ontology + "%3B+dct%3Asubject+dbc%3A" + category + "%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"

	//fmt.Println("Querying DBPedia with: " + Url.String())

	// create json from json-string answer
	var data = getJSON(Url.String()).Get("results", "bindings").GetArray()

	// convert json structure to go-objects array (information array)
	var res []info
	for _, d := range data {

		var t = string(d.Get("s1").GetStringBytes("type"))
		var v = string(d.Get("s1").GetStringBytes("value"))
		res = append(res, info{t, v})
	}

	return res

}

// TODO
/*

- filter out parenthesis in queries
- get siblings of classes
- get numeric properties


*/

/// CODE GRAVEYARD

//class = append(class, filterInfo(classQuery, []string{"rdf-syntax-ns#type"}, []string{"dbpedia.org/ontology"})...)

//var superClasses = filterInfo(res, []string{"rdf-syntax-ns#type"}, []string{"dbpedia.org/ontology"})
//var subjects = filterInfo(queryDBP(cleanURL(class[0].val.val)), []string{"terms/subject"}, []string{})

//var siblingClasses []info

//for _, sub := range subjects {

//var sc = queryDBPSiblings(cleanURL(class[0].val.val), cleanCategory(sub.val.val))
//var sc = queryDBPSiblings("Bridge", cleanCategory(sub.val.val))

//filterInfo(queryDBP(cleanURL(sub.val.val)), []string{"terms/subject"})
//siblingClasses = append(siblingClasses, sc...)

//}

//fmt.Println("test")
//return ItemData{class[0], superClasses, siblingClasses, subjects, superClasses}
//return ItemData{class, superClasses}

// get siblings from dbpedia sparql point to generate distractors

//select ?s1
//where {
//  ?s1 a dbo:Bridge; dct:subject dbc:Bridges
//}
