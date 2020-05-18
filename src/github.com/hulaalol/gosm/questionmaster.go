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

var stopwords = [...]string{"street", "road", "highway", "way", "avenue", "strait", "drive", "lane", "grove", "gardens", "place", "circus", "crescent", "bypass", "close", "square", "hill", "mews", "vale", "rise", "row", "mead", "wharf", "walk"}

//var stopwordsGER = [...]string{"weg", "straße", "strasse", "allee", "gasse", "Straße", "Weg", "Strasse", "Allee"}
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
		s = strings.ReplaceAll(s, strings.Title(sw), "")
	}

	/*
		for _, sy := range syllables {

			var syLength = len(sy)
			var sLength = len(s)

			if strings.Compare(s[sLength-1-syLength:sLength], sy) == 1 {
				s = s[0 : sLength-1-syLength]
			}
		}
	*/
	return s
}

func cleanURL(s string) string {

	// resource
	var dbr = "http://dbpedia.org/resource/"
	if strings.Contains(s, dbr) {
		return strings.Replace(s, dbr, "", -1)
	}

	// item with slash e.g. "HIV/AIDS"
	if !strings.Contains(s, ":") {
		return s
	}

	// URL

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
	t = strings.ReplaceAll(t, "/", "\\/")

	return t
}

type Question struct {
	item   string
	answer string
	d1     string
	d2     string
	d3     string
}

type ItemData struct {
	item           string
	class          string
	superClass     string
	siblingClasses []info
	siblings       []info
	supersiblings  []info
	categories     []info
	categoriesExp  [][]info
	catDists       []catDist
	properties     []information
	propertyDists  []propDist
	depiction      string
}

func genItem(item string, getPropDists bool) ItemData {

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

		// TODO: catch weird dbpedia classes
		var className []string
		if item == "England" {
			className = []string{"Country", "dbo"}
		} else {
			className = getClassName(res)
		}

		if className[0] == "null" {
			// this is not an entity --> search for disambiguations

			var disambiguations = getDisambiguations(item)

			if len(disambiguations) > 0 {
				var s = rand.NewSource(time.Now().Unix())
				var r = rand.New(s) // initialize local pseudorandom generator
				var newItem = cleanURL(disambiguations[r.Intn(len(disambiguations))].val)

				return genItem(newItem, getPropDists)
			} else {
				fmt.Println("cant generate items, error")
				return genEmptyItem()
			}

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

		var p []propDist
		if getPropDists {
			p = getPropDistractor(props)
		} else {
			p = []propDist{}

		}

		return ItemData{item, cN, superClass[0], getSiblingClasses(superClass[0], cN), getSiblings(cN), getSiblings(superClass[0]), cats, catsExp, catDists, props, p, getDepiction(item)}

	} else {
		return genItem(cleanURL(redirect), getPropDists)
	}

}

func genEmptyItem() ItemData {
	return ItemData{"null", "null", "null", []info{}, []info{}, []info{}, []info{}, [][]info{}, []catDist{}, []information{}, []propDist{}, "null"}
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

type propDist struct {
	property    string
	answer      string
	distractors []info
}

func getPropDistractor(props []information) []propDist {

	// get only one item
	rand.Seed(time.Now().UnixNano())

	p := rand.Perm(len(props))
	dest := make([]information, len(props))

	for i, v := range p {
		dest[v] = props[i]
	}

	var res = []propDist{}
	for _, p := range dest {
		if strings.Contains(p.val.val, "/dbpedia.org/resource/") {
			//fetch resource and get distractors

			var resource = cleanURL(p.val.val)
			var resourceItem = genItem(resource, false)

			// check if there are at least 3 distractors
			if len(resourceItem.siblings) > 2 {
				res = append(res, propDist{p.typ.val, resource, resourceItem.siblings})
				break
			}

		}

	}
	return res
}

func getPropDistractors(props []information) []propDist {

	var res = []propDist{}
	for _, p := range props {
		if strings.Contains(p.val.val, "/dbpedia.org/resource/") {
			//fetch resource and get distractors

			var resource = cleanURL(p.val.val)
			var resourceItem = genItem(resource, false)

			res = append(res, propDist{p.typ.val, resource, resourceItem.siblings})
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

		fmt.Println("no redirect - does res already contain info? - catch disambiguation")

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
	var classRDF = filterInfo(queryResult, []string{"rdf-syntax-ns#type", "rdf-schema#subClassOf"}, []string{"dbpedia.org/ontology", "owl#Class", "owl#Thing"})
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

	//filter classnames
	var filter = [...]string{"http://dbpedia.org/ontology/Location", "http://dbpedia.org/ontology/Place", "http://dbpedia.org/ontology/Agent", "http://dbpedia.org/ontology/Person", "http://dbpedia.org/ontology/Place", "http://www.w3.org/2002/07/owl#Thing"}

	var fClass []information
	for _, c := range class {

		var skip = false
		for _, f := range filter {

			if strings.Contains(c.val.val, f) {
				skip = true
				continue
			}
		}

		if !skip {
			fClass = append(fClass, c)
		}
	}

	var className = ""
	if len(fClass) > 0 {
		className = fClass[0].val.val
	} else {
		// TODO: maybe prioritize the first class found
		className = class[0].val.val
	}

	var typ string

	if strings.Contains(className, "ontology") {
		typ = "dbo"
	} else {
		typ = "dbr"
	}

	return []string{cleanURL(className), typ}
}

func getSiblings(class string) []info {

	// dbo
	fmt.Println("getting siblings for: " + class)
	class = cleanSpecialCharacters(class)
	var rq = "default-graph-uri=http://dbpedia.org&query=PREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0APREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0ASELECT+%3Fproperty%0D%0AWHERE+%7B+++++++%0D%0A++++++++%3Fproperty+dbo%3Atype+res%3A" + class + "+++%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)
	var dbo = json2infoArray(data)

	//rdf
	rq = "default-graph-uri=http://dbpedia.org&query=PREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0APREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0APREFIX+rdf%3A+%3Chttp%3A%2F%2Fwww.w3.org%2F1999%2F02%2F22-rdf-syntax-ns%23%3E%0D%0ASELECT+%3Fproperty%0D%0AWHERE+%7B%0D%0A+++++++++%3Fproperty+rdf%3Atype+dbo%3A" + class + "+%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	data = query(rq)
	var rdf = json2infoArray(data)

	return append(dbo, rdf...)
}

func getSiblingClasses(superClass string, class string) []info {
	/*
	   PREFIX dbo: <http://dbpedia.org/ontology/>
	   PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

	   SELECT ?property WHERE {
	   	   	 ?property rdfs:subClassOf dbo:Person
	   	   }
	*/

	fmt.Println("getting sibling classes for superclass " + superClass)
	superClass = cleanSpecialCharacters(superClass)
	var rq = "default-graph-uri=http://dbpedia.org&query=PREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0APREFIX+rdfs%3A+%3Chttp%3A%2F%2Fwww.w3.org%2F2000%2F01%2Frdf-schema%23%3E%0D%0A%0D%0ASELECT+%3Fproperty+WHERE+%7B%0D%0A%09+++%09+%3Fproperty+rdfs%3AsubClassOf+dbo%3A" + superClass + "%0D%0A%09+++%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)

	var d = json2infoArray(data)

	// filter class as it is not sibling of itself
	var res = []info{}
	for _, v := range d {

		if cleanURL(v.val) != class {

			res = append(res, v)
		}

	}
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

func getDepiction(item string) string {
	fmt.Println("getting depiction of " + item)
	item = cleanSpecialCharacters(item)
	var rq = "default-graph-uri=http://dbpedia.org&query=PREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0APREFIX+foaf%3A%3Chttp%3A%2F%2Fxmlns.com%2Ffoaf%2F0.1%2F%3E%0D%0ASELECT+%3Fproperty+WHERE+%7B%0D%0A%09+++%09+res%3A" + item + "+foaf%3Adepiction+%3Fproperty%0D%0A%09+++%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)
	var res = json2infoArray(data)

	if res == nil {
		return "null"
	} else {
		return res[0].val
	}
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
					// filter if there are too few distractors
					if len(distractors) > 2 {
						res = append(res, catDist{cat.val, distractors})
					}
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
	//fmt.Println("Querying DBPedia with: " + Url.String())

	return getJSON(Url.String()).Get("results", "bindings").GetArray()
}

func questionCatDist(d ItemData) Question {

	if d.catDists != nil && len(d.catDists) > 0 {

		// pick random cat
		var s = rand.NewSource(time.Now().Unix())
		var r = rand.New(s) // initialize local pseudorandom generator
		catD := d.catDists[r.Intn(len(d.catDists))]

		var question = d.item + " is a ..."
		var answer = cleanURL(catD.answer)

		rand.Seed(time.Now().UnixNano())

		dest := make([]string, len(catD.distractors))
		perm := rand.Perm(len(catD.distractors))

		for i, v := range perm {
			dest[v] = catD.distractors[i]
		}

		var d1 = cleanURL(dest[0])
		var d2 = cleanURL(dest[1])
		var d3 = cleanURL(dest[2])
		return Question{question, answer, d1, d2, d3}

	} else {
		return Question{"null", "null", "null", "null", "null"}
	}

}

func questionPropDist(d ItemData) Question {

	if d.propertyDists != nil && len(d.propertyDists) > 0 {

		var s = rand.NewSource(time.Now().Unix())
		var r = rand.New(s)
		propD := d.propertyDists[r.Intn(len(d.propertyDists))]

		var question = d.item + " is " + propD.property + " ..."
		var answer = propD.answer

		rand.Seed(time.Now().UnixNano())

		dest := make([]info, len(propD.distractors))
		perm := rand.Perm(len(propD.distractors))
		for i, v := range perm {
			dest[v] = propD.distractors[i]
		}

		var d1 = cleanURL(dest[0].val)
		var d2 = cleanURL(dest[1].val)
		var d3 = cleanURL(dest[2].val)

		return Question{question, answer, d1, d2, d3}

	} else {
		return getEmptyQuestion()
	}

}

func questionSiblingClasses(d ItemData) Question {

	var question = d.item + " is a ..."
	var answer = d.class

	// check if siblings are present
	if d.siblingClasses != nil && len(d.siblingClasses) > 2 {

		// shuffle siblings
		rand.Seed(time.Now().UnixNano())

		dest := make([]info, len(d.siblingClasses))
		perm := rand.Perm(len(d.siblingClasses))
		for i, v := range perm {
			dest[v] = d.siblingClasses[i]
		}

		var d1 = cleanURL(dest[0].val)
		var d2 = cleanURL(dest[1].val)
		var d3 = cleanURL(dest[2].val)

		return Question{question, answer, d1, d2, d3}
	} else {
		return getEmptyQuestion()
	}

}

func questionPropLiteral(d ItemData) Question {

	// select property

	if d.properties == nil || len(d.properties) == 0 {
		return getEmptyQuestion()
	} else {
		rand.Seed(time.Now().UnixNano())
		//var s = rand.NewSource(time.Now().Unix())
		//var r = rand.New(s) // initialize local pseudorandom generator
		var idx = rand.Intn(len(d.properties))
		var p = d.properties[idx]
		//cleanURL(disambiguations[r.Intn(len(disambiguations))].val)

		// get distractors

		var question = "The property " + p.typ.val + " of " + d.item + " is ..."
		var answer = p.val.val

		fmt.Println("getting property distractors for property " + p.typ.val)
		var rq = "default-graph-uri=http://dbpedia.org&query=%0D%0A%0D%0APREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0A%0D%0ASELECT++%3Fp+%3Fproperty+WHERE+%7B%0D%0A%09+++%09++%3Fp+a+dbo%3A" + d.superClass + "+%3B+%3C" + p.typ.val + "%3E%3Fproperty+.++%0D%0A%09+++%7D%0D%0ALIMIT+500&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
		var data = query(rq)
		var res = json2infoArray(data)

		if res == nil {
			return getEmptyQuestion()
		}

		rand.Seed(time.Now().UnixNano())

		dest := make([]info, len(res))
		perm := rand.Perm(len(res))
		for i, v := range perm {
			dest[v] = res[i]
		}

		var d1 = cleanURL(dest[0].val)
		var d2 = cleanURL(dest[1].val)
		var d3 = cleanURL(dest[2].val)

		return Question{question, answer, d1, d2, d3}

	}

}

func genQuestion(item string) []Question {

	item = cleanStreetname(item)
	item = strings.TrimSpace(item)

	item = strings.ReplaceAll(item, " ", "_")

	fmt.Println("trying to generate a question for: " + item)
	var d = genItem(item, true)

	for d.item == "null" {
		item = item[:len(item)-1]
		d = genItem(item, true)
	}

	return []Question{questionCatDist(d), questionSiblingClasses(d), questionPropLiteral(d), questionPropLiteral(d), questionPropLiteral(d), questionPropLiteral(d), questionPropLiteral(d)}
}

func getEmptyQuestion() Question {
	return Question{"null", "null", "null", "null", "null"}

}

// TODO
/*

- filter out parenthesis and commas in queries DONE
- get siblings of classes DONE
- get categories DONE
- failsafe for redirects! DONE
- select from multiple entries (Morpeth --> is a school, a place, a band etc.) http://dbpedia.org/ontology/wikiPageDisambiguates DONE
- get numeric properties --> ez DONE

- find property distractors (e.g. Stanford University --> private university class --> is dbo:type of) DONE!
- if no siblings, find siblings of parent class DONE --> supersiblings!


- create hardcoded list of dbo:country!
- get depiction (embed html link to image)
- generate Questions YAY :)


- compare property to siblings (find 4 with same property) POSTPONE

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
