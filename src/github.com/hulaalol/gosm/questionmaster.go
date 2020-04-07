package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/valyala/fastjson"
)

var stopwords = [...]string{"weg", "straße", "strasse", "allee", "gasse", "Straße", "Weg", "Strasse", "Allee"}
var syllables = [...]string{"er"}

var propBlacklist = [...]string{"rdf-syntax-ns#type", "wikiPageRevisionID", "owl#sameAs", "rdf-schema#comment", "rdf-schema#label", "#wasDerivedFrom", "hypernym", "depiction", "wikiPageExternalLink", "wikiPageID", "subject", "isPrimaryTopicOf", "thumbnail", "abstract",
	"caption", "property/name", "/foaf/0.1/name", "ontology/picture", "ontology/type", "dbpedia.org/property/id", "property/imageSize", "/property/title", "property/wordnet_type", "/property/note", "/property/servingSize", "/property/sourceUsda", "staticImage"}

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

	if lastSlash == 0 {
		return s
	} else {
		return s[lastSlash+1 : len(s)]
	}

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

func cleanSpecialCharacters(s string) string {

	var t = strings.ReplaceAll(s, "(", "\\(")
	t = strings.ReplaceAll(t, ")", "\\)")
	t = strings.ReplaceAll(t, ",", "\\,")

	return t
}

type Question struct {
	item        string
	answer      string
	distractors [3]string
}

type ItemData struct {
	item           string
	class          string
	superClasses   string
	siblingClasses []info
	categories     []info
	categoriesExp  [][]info
	catDists       []catDist
	subjects       string
	properties     []information
}

func genItem(item string) ItemData {

	// clean
	// check redirects
	// check ambiguates

	var redirect = getRedirect(item)

	if redirect == "null" {

		var i = cleanSpecialCharacters(item)

		var res = queryDBP(i, "dbr")
		if len(res) == 0 {
			res = queryDBP(i, "dbo")
		}

		var className = getClassName(res)

		if className[0] == "null" {
			// this is not an entity --> search for disambiguations

			var disambiguations = getDisambiguations(item)

			var s = rand.NewSource(time.Now().Unix())
			var r = rand.New(s) // initialize local pseudorandom generator
			var newItem = cleanURL(disambiguations[r.Intn(len(disambiguations))].val)

			return genItem(newItem)
		}
		//fmt.Println("Found class " + className[0] + " for " + item)

		// find superclasses and siblings

		var cN = className[0]
		className[0] = cleanSpecialCharacters(className[0])

		var classQuery = queryDBP(className[0], className[1])
		var superClass = getClassName(classQuery)
		//fmt.Println("Found superclass " + superClass[0] + " for " + item)

		var cats = getCategories(cN)

		var catsExp [][]info
		for _, c := range cats {
			var catX = expandCategory(cleanURL(c.val))
			catsExp = append(catsExp, catX)
		}

		var catDists = getCategoryDistractors(cats, catsExp)

		var props = getProps(res)

		return ItemData{item, cN, superClass[0], getSiblings(cN), cats, catsExp, catDists, "", props}

	} else {
		return genItem(cleanURL(redirect))
	}

}

func getProps(data []information) []information {

	var res = []information{}
	for _, d := range data {

		var skip = false
		for _, bl := range propBlacklist {
			if strings.Contains(d.typ.val, bl) {
				skip = true
			}
		}

		if skip {
			continue
		} else {
			res = append(res, d)
		}

	}
	return res
}

func queryDBP(item string, typ string) []information {
	// create json from json-string answer
	var rq = "default-graph-uri=http://dbpedia.org&query=select+distinct+?property+?value%7B%0D%0A++" + typ + "%3A" + item + "+%3Fproperty+%3Fvalue%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)
	//fmt.Println("Found " + strconv.Itoa(len(res)) + " informations about " + item)

	var res = json2informationArray(data)

	var redirect = getRedirect(item)
	if redirect != "null" {
		fmt.Println("Found redirect. Following...")
		res = queryDBP(redirect, typ)
	} else {

		//no redirect - does res already contain info?
		// catch disambiguation

	}

	return res
}

func getDisambiguations(item string) []info {
	/*	PREFIX res: <http://dbpedia.org/resource/>
		PREFIX dbo: <http://dbpedia.org/ontology/>

		SELECT ?property WHERE {
				  res:Morpeth dbo:wikiPageDisambiguates ?property
		}

		https://dbpedia.org/sparql?default-graph-uri=http%3A%2F%2Fdbpedia.org&query=PREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0APREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0A%0D%0ASELECT+%3Fproperty+WHERE+%7B%0D%0A%09++++++res%3AMorpeth+dbo%3AwikiPageDisambiguates+%3Fproperty%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+
	*/
	fmt.Println("getting disambiguations for: " + item)
	item = cleanSpecialCharacters(item)
	var rq = "default-graph-uri=http%3A%2F%2Fdbpedia.org&query=PREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0APREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0A%0D%0ASELECT+%3Fproperty+WHERE+%7B%0D%0A%09++++++res%3A" + item + "+dbo%3AwikiPageDisambiguates+%3Fproperty%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)
	var res = json2infoArray(data)
	return res
}

func getRedirect(item string) string {
	/*
		PREFIX res: <http://dbpedia.org/resource/>
		PREFIX dbo: <http://dbpedia.org/ontology/>
		SELECT ?property WHERE {
		    res:Arch_Bridge dbo:wikiPageRedirects ?property
		}

		https://dbpedia.org/sparql?default-graph-uri=http%3A%2F%2Fdbpedia.org&query=PREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0APREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0A%0D%0ASELECT+%3Fproperty+WHERE+%7B+%0D%0A+++res%3AArch_Bridge+dbo%3AwikiPageRedirects+%3Fproperty%0D%0A%7D++%0D%0A&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+
	*/
	fmt.Println("getting redirect for: " + item)
	item = cleanSpecialCharacters(item)
	var rq = "default-graph-uri=http%3A%2F%2Fdbpedia.org&query=PREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0APREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0A%0D%0ASELECT+%3Fproperty+WHERE+%7B+%0D%0A+++res%3A" + item + "+dbo%3AwikiPageRedirects+%3Fproperty%0D%0A%7D++%0D%0A&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)

	if data == nil {
		// there is no redirect
		return "null"
	} else {
		// convert json structure to go-objects array (information array)
		var res = cleanURL(string(data[0].Get("property").GetStringBytes("value")))
		return res
	}

}

func getClassName(queryResult []information) []string {
	// determine the proper class of the item
	var classRDF = filterInfo(queryResult, []string{"rdf-syntax-ns#type", "rdf-schema#subClassOf"}, []string{"dbpedia.org/ontology", "owl#Class"})
	var classDBO = filterInfo(queryResult, []string{"dbpedia.org/ontology/type", "rdf-schema#subClassOf"}, []string{"dbpedia.org"})
	var class []information
	if len(classDBO) == 0 && len(classRDF) == 0 {

		//check for yago class
		// e.g. : http://dbpedia.org/class/yago/WikicatExtinctEarldomsInThePeerageOfEngland
		var classOther = filterInfo(queryResult, []string{"rdf-syntax-ns#type", "rdf-schema#subClassOf"}, []string{"dbpedia.org/class/"})

		if len(classOther) == 0 {
			fmt.Println("Could not find class!")
			return []string{"null"}
		} else {
			return []string{cleanURL(classOther[0].val.val), "other"}
		}

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

func getSiblings(class string) []info {

	fmt.Println("getting siblings for: " + class)
	class = cleanSpecialCharacters(class)
	var rq = "default-graph-uri=http://dbpedia.org&query=PREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0APREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0ASELECT+%3Fproperty%0D%0AWHERE+%7B+++++++%0D%0A++++++++%3Fproperty+dbo%3Atype+res%3A" + class + "+++%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)
	var res = json2infoArray(data)

	return res
}

func getCategories(class string) []info {

	fmt.Println("getting categories for: " + class)
	class = cleanSpecialCharacters(class)
	var rq = "default-graph-uri=http://dbpedia.org&query=PREFIX++dct%3A++%3Chttp%3A%2F%2Fpurl.org%2Fdc%2Fterms%2F%3E+%0D%0APREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0A%0D%0ASELECT+%3Fproperty+WHERE+%7B+%0D%0A+++res%3A" + class + "+dct%3Asubject+%3Fproperty%0D%0A%7D++%0D%0AORDER+BY+%3Fproperty&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)

	if data == nil {
		var redirect = getRedirect(class)

		if redirect == "null" {
			return []info{}
		} else {
			return getCategories(redirect)
		}
	}

	var res = json2infoArray(data)
	return res
}

func expandCategory(category string) []info {
	/*
	   PREFIX res: <http://dbpedia.org/resource/>
	   SELECT ?property WHERE {
	   	?property skos:broader res:Category:Arch_bridges
	   }
	   https://dbpedia.org/sparql?default-graph-uri=http%3A%2F%2Fdbpedia.org&query=PREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0ASELECT+%3Fproperty+WHERE+%7B%0D%0A%09+++++++++%3Fproperty+skos%3Abroader+res%3ACategory%3AArch_bridges++%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+
	*/

	fmt.Println("expanding category " + category)
	var rq = "default-graph-uri=http%3A%2F%2Fdbpedia.org&query=PREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0ASELECT+%3Fproperty+WHERE+%7B%0D%0A%09+++++++++%3Fproperty+skos%3Abroader+res%3A" + category + "++%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)
	var res = json2infoArray(data)
	return res

}

type catDist struct {
	answer      string
	distractors []string
}

func getCategoryDistractors(cats []info, catsEx [][]info) []catDist {

	var res = []catDist{}
	for _, cat := range cats {
		for _, catzEx := range catsEx {
			for _, cex := range catzEx {
				if cat.val == cex.val {
					// we have a match
					// get distractors
					var distractors = []string{}
					for _, catString := range catzEx {
						if catString.val != cat.val {
							distractors = append(distractors, catString.val)
						}
					}
					res = append(res, catDist{cat.val, distractors})
					continue
				}
			}
		}
	}
	return res

}

func json2infoArray(data []*fastjson.Value) []info {
	// convert json structure to go-objects array (info array)
	var res []info
	for _, d := range data {
		var p = info{string(d.Get("property").GetStringBytes("type")), string(d.Get("property").GetStringBytes("value"))}
		res = append(res, p)
	}
	return res
}

func json2informationArray(data []*fastjson.Value) []information {
	// convert json structure to go-objects array (information array)
	var res []information
	for _, d := range data {

		var p = info{string(d.Get("property").GetStringBytes("type")), string(d.Get("property").GetStringBytes("value"))}
		var v = info{string(d.Get("value").GetStringBytes("type")), string(d.Get("value").GetStringBytes("value"))}
		res = append(res, information{p, v})
	}
	return res
}

func query(rawquery string) []*fastjson.Value {
	Url, err := url.Parse("https://dbpedia.org")
	if err != nil {
		panic("boom")
	}

	Url.Path += "/sparql"
	Url.RawQuery = rawquery
	fmt.Println("Querying DBPedia with: " + Url.String())

	return getJSON(Url.String()).Get("results", "bindings").GetArray()
}

// TODO
/*

- filter out parenthesis and commas in queries DONE
- get siblings of classes DONE
- get categories DONE
- failsafe for redirects! DONE
- select from multiple entries (Morpeth --> is a school, a place, a band etc.) http://dbpedia.org/ontology/wikiPageDisambiguates DONE
- get numeric properties --> ez DONE

- find property distractors (e.g. Stanford University --> private university class --> is dbo:type of)
- if no siblings, find siblings of parent class
- compare property to siblings (find 4 with same property)
- get depiction (embed html link to image)

Question generation
- manipulate numbers (*1.05) i.e.




// QUERIES
PREFIX dbo: <http://dbpedia.org/ontology/>
PREFIX res: <http://dbpedia.org/resource/>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
SELECT ?s
WHERE {
        ?s dbo:type res:Community_school_\(England_and_Wales\)
}



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

/* func queryDBPSiblings(ontology string, category string) []info {

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
*/
