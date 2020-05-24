package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/valyala/fastjson"
)

var stopwords = [...]string{"'s", " street", " road", " Highway", "highway ", "way", " avenue", " drive", " lane", " circus", " Row", "the ", " place", "grove", "pavement"}

//var stopwords = [...]string{"'s", " street", " road", " highway", "highway ", "way", " avenue", "strait", "drive", " lane", "grove", "gardens", "place", "circus", "crescent", "bypass", "close", "square", "hill", "mews", "vale", "rise", " Row", "mead", "wharf", "walk", "the "}
var stopwordsItem = [...]string{"great", "the", "bridge", "high", "st", "court", "mews", "square", "end", "new", "alley", "upper", "lower"}

//var stopwordsGER = [...]string{"weg", "straße", "strasse", "allee", "gasse", "Straße", "Weg", "Strasse", "Allee"}
var syllables = [...]string{"er"}
var randlim = "%0D%0A%09%09ORDER+BY+RAND%28%29%0D%0A%09%09limit+100"

var propBlacklist = [...]string{"rdf-syntax-ns#type", "wikiPageRevisionID", "owl#sameAs", "rdf-schema#comment", "rdf-schema#label", "#wasDerivedFrom", "hypernym", "depiction", "wikiPageExternalLink", "wikiPageID", "subject", "isPrimaryTopicOf", "thumbnail", "abstract",
	"caption", "commons", "seconded", "width", "wikt", "alt", "expiry", "wikititle", "list", "gridReference", "pushLabelPosition", "small", "urlname", "annotFontSize", "/property/image", "property/align", "property/footnotes", "property/footer", "imageCaption", "labelPosition", "locatorMap", "popRefCbs", "differentFrom", "popRefName", "rdf-schema#seeAlso", "property/longEw", "foaf/0.1/homepage", "property/name", "/foaf/0.1/name", "ontology/picture", "ontology/type", "dbpedia.org/property/id", "property/imageSize", "/property/title", "property/wordnet_type", "/property/note", "/property/servingSize", "/property/sourceUsda", "staticImage", "/georss/point", "/geo/wgs84", "/ontology/wikiPageDisambiguates"}

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

	var owlPrefix = "http://www.w3.org/2002/07/owl#"
	if strings.Contains(s, owlPrefix) {
		return strings.Replace(s, owlPrefix, "", -1)
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

	// new escape underscore in queries
	t = strings.ReplaceAll(t, "_", "\\_")

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
	item            string
	class           string
	superClass      string
	siblingClasses  []info
	siblings        []info
	supersiblings   []info
	properties      []information
	propertyDists   []propDist
	depiction       string
	abstract        string
	disambiguations []string
}

func genItem(item string, getPropDists bool, followDisambiguations bool) ItemData {

	fmt.Println("generating item " + item)

	// query
	var res = queryDBP(item, "dbr")
	if len(res) == 0 {
		fmt.Println("dbr returned no results for " + item + "! Querying dbo now...")
		res = queryDBP(item, "dbo")
	}

	// check disambiguaty, check hypernym, check redirect
	if getPropRes("http://www.w3.org/1999/02/22-rdf-syntax-ns#type", res).val.val == "null" {
		// check for redirects
		var redirect = getRedirectRes(res)
		if redirect != "null" {
			fmt.Println("following redirect " + redirect + " for item " + item)
			return genItem(redirect, getPropDists, true)
		}
	}

	// if no results, try to split words
	if res == nil {
		if strings.Contains(item, "_") {
			words := strings.Split(item, "_")

			rand.Seed(time.Now().UnixNano())
			rand.Shuffle(len(words), func(i, j int) { words[i], words[j] = words[j], words[i] })

			for _, w := range words {

				strings.ReplaceAll(w, ",", "")

				if strings.Contains(w, ")") || strings.Contains(w, "(") {
					continue
				}

				var isStopword = false
				for _, sw := range stopwordsItem {
					if w == sw || w == strings.Title(sw) {
						isStopword = true
					}
				}

				if !isStopword {
					return genItem(w, getPropDists, true)
				}
			}
		}
		return genEmptyItem()
		//return genItem(item[:len(item)-1], getPropDists)
	}

	// get Class
	var className []string

	// catch weird dbpedia classes
	className = getClassName(res)

	// get properties
	var props = getProps(res)

	// if no class and no properties, search for disambiguations and hypernyms
	//if className[0] == "null" && len(props) == 0 {
	//if len(props) == 0 && followDisambiguations {

	var disambiguations = getDisambiguationsResAll(res)

	if len(props) == 0 && className[0] == "null" && followDisambiguations {
		fmt.Println("no class and no props, searching for disambiguations of " + item + "...")

		/*
			var disambiguation = getDisambiguationsRes(res)

			if disambiguation != "null" {
				return genItem(disambiguation, getPropDists)

		*/

		//TODO: shuffle disambiguations
		rand.Shuffle(len(disambiguations), func(i, j int) { disambiguations[i], disambiguations[j] = disambiguations[j], disambiguations[i] })

		for _, d := range disambiguations {
			if d != "null" {
				var dItem = genItem(d, getPropDists, false)
				if dItem.item != "null" {
					return dItem
				}
			}
		}

		//} else {
		fmt.Println("no disambiguations, searching for hypernyms of " + item + "...")

		var hypernym = getHypernymsRes(res)
		if hypernym != "null" {
			return genItem(hypernym, getPropDists, false)
		}
		//}

		fmt.Println("no disambiguations and hypernyms, trying to generate props for " + item + "...")
		var p = getProps(res)
		return ItemData{item, "null", "null", []info{}, []info{}, []info{}, p, getPropDistractor(p), getDepiction(res), getAbstract(res), disambiguations}
	}

	// get superclass
	var cN string = className[0]
	//var classQuery []information = []information{}
	var superClass []string = []string{"null"}

	if className[0] != "null" && getPropDists {
		//cN = className[0]
		//className[0] = cleanSpecialCharacters(className[0])

		//classQuery = queryDBP(className[0], className[1])
		//superClass = getClassName(classQuery)
		var sc = getSuperClass(cN)
		superClass = []string{sc[0].val, sc[0].typ}
	}

	// get a property distractor
	var p = []propDist{}

	if len(props) != 0 {
		if getPropDists {
			p = getPropDistractor(props)
		} else {
			p = []propDist{}
		}
	}

	var siblingClasses = []info{}
	var classSiblings = []info{}
	var superClassSiblings = []info{}

	if superClass[0] != "null" && cN != "null" {
		siblingClasses = getSubClasses(superClass[0])

		var s = make([]info, 0)
		for _, sibling := range siblingClasses {
			if cN != sibling.val {
				s = append(s, sibling)
			}
		}
		siblingClasses = s
		//siblingClasses = getSiblingClasses(superClass[0], cN)
	}

	//if cN != "null" && !getPropDists {
	//	classSiblings = getSiblings(cN)
	if cN != "null" {
		classSiblings = getClassMembers(cN)
	}

	return ItemData{item, cN, superClass[0], siblingClasses, classSiblings, superClassSiblings, props, p, getDepiction(res), getAbstract(res), disambiguations}
}

func genEmptyItem() ItemData {
	return ItemData{"null", "null", "null", []info{}, []info{}, []info{}, []information{}, []propDist{}, "null", "null", []string{}}
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

		if !skip {
			if strings.Contains(d.typ.val, "property") {
				var propName = cleanURL(d.typ.val)
				if len(propName) <= 2 {
					skip = true
				}
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
	var superRes = []propDist{}
	for _, p := range dest {

		if strings.Contains(p.val.val, "/dbpedia.org/resource/") {
			//fetch resource and get distractors

			fmt.Println("get Prop distractor for resource property " + p.typ.val)

			var resource = cleanURL(p.val.val)
			var resourceItem = genItem(resource, false, false)

			// check if there are at least 3 distractors
			if len(resourceItem.siblings) > 2 {
				res = append(res, propDist{p.typ.val, resource, resourceItem.siblings})
				return res
			}

			if len(resourceItem.siblingClasses) > 2 {
				superRes = append(superRes, propDist{resourceItem.superClass, resourceItem.class, resourceItem.siblingClasses})
			}
		}
	}

	if len(superRes) > 0 {
		rand.Seed(time.Now().UnixNano())
		var idx = rand.Intn(len(superRes))
		return []propDist{superRes[idx]}
	} else {
		return res
	}
}

func getPropDistractors(props []information) []propDist {

	var res = []propDist{}
	for _, p := range props {
		if strings.Contains(p.val.val, "/dbpedia.org/resource/") {
			//fetch resource and get distractors

			var resource = cleanURL(p.val.val)
			var resourceItem = genItem(resource, false, false)

			res = append(res, propDist{p.typ.val, resource, resourceItem.siblings})
		}

	}
	return res
}

func queryDBP(item string, typ string) []information {
	var i = cleanSpecialCharacters(item)

	// create json from json-string answer
	var rq = "default-graph-uri=http://dbpedia.org&query=select+distinct+?property+?value%7B%0D%0A++" + typ + "%3A" + i + "+%3Fproperty+%3Fvalue%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)

	return json2informationArray(data)
}

func getPropResAll(prop string, res []information) []information {
	var results []information = make([]information, 0)

	for _, r := range res {
		if r.typ.val == prop {
			results = append(results, r)
		}
	}
	if len(results) == 0 {
		return []information{information{info{"null", "null"}, info{"null", "null"}}}
	} else {
		return results
	}
}

func getPropRes(prop string, res []information) information {

	var results []information = make([]information, 0)

	for _, r := range res {
		if r.typ.val == prop {
			results = append(results, r)
		}
	}

	if len(results) == 0 {
		return information{info{"null", "null"}, info{"null", "null"}}
	}

	var s = rand.NewSource(time.Now().Unix())
	var r = rand.New(s) // initialize local pseudorandom generator
	d := results[r.Intn(len(results))]

	return d

}

func getDisambiguationsResAll(res []information) []string {

	var dis = getPropResAll("http://dbpedia.org/ontology/wikiPageDisambiguates", res)

	var result []string = make([]string, 0)

	for _, d := range dis {
		result = append(result, cleanURL(d.val.val))
	}
	return result
}

func getDisambiguationsRes(res []information) string {
	return cleanURL(getPropRes("http://dbpedia.org/ontology/wikiPageDisambiguates", res).val.val)
}

func getHypernymsRes(res []information) string {
	return cleanURL(getPropRes("http://purl.org/linguistics/gold/hypernym", res).val.val)
}

func getRedirectRes(res []information) string {

	var rds []information = make([]information, 0)

	for _, r := range res {
		if r.typ.val == "http://dbpedia.org/ontology/wikiPageRedirects" {
			rds = append(rds, r)
		}
	}

	if len(rds) == 0 {
		return "null"
	}

	var s = rand.NewSource(time.Now().Unix())
	var r = rand.New(s) // initialize local pseudorandom generator
	rd := rds[r.Intn(len(rds))]

	return cleanURL(rd.val.val)

}

func getSubClasses(class string) []info {
	/*
		SELECT ?property WHERE {
					?property <http://www.w3.org/2000/01/rdf-schema#subClassOf> <http://dbpedia.org/class/yago/Goddess109535622>
		}
		ORDER BY RAND()
		LIMIT 10
	*/

	var rq = "default-graph-uri=http://dbpedia.org&query=SELECT+%3Fproperty+WHERE+%7B%0D%0A%3Fproperty+%3Chttp%3A%2F%2Fwww.w3.org%2F2000%2F01%2Frdf-schema%23subClassOf%3E+<" + class + ">%0D%0A%7D%0D%0AORDER+BY+RAND%28%29%0D%0ALIMIT+100%0D%0A%0D%0A%0D%0A&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)
	var res = json2infoArray(data)

	return res

}

func getClassMembers(class string) []info {
	/*
		SELECT ?property WHERE {
		?property <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://dbpedia.org/class/yago/Goddess109535622>
		}
		ORDER BY RAND()
		LIMIT 10
	*/
	var rq = "default-graph-uri=http://dbpedia.org&query=SELECT+%3Fproperty+WHERE+%7B%0D%0A%3Fproperty+%3Chttp%3A%2F%2Fwww.w3.org%2F1999%2F02%2F22-rdf-syntax-ns%23type%3E+<" + class + ">%0D%0A%7D%0D%0AORDER+BY+RAND%28%29%0D%0ALIMIT+100%0D%0A%0D%0A%0D%0A&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)
	var res = json2infoArray(data)
	return res
}

func getSuperClass(class string) []info {
	/*
		PREFIX dbo: <http://dbpedia.org/ontology/>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

		SELECT ?property WHERE {
			   	   	 <http://dbpedia.org/class/yago/WikicatRomanGoddesses> rdfs:subClassOf ?property
		}
	*/
	var rq = "default-graph-uri=http://dbpedia.org&query=SELECT+%3Fproperty+WHERE+%7B%0D%0A<" + class + ">+%3Chttp%3A%2F%2Fwww.w3.org%2F2000%2F01%2Frdf-schema%23subClassOf%3E+%3Fproperty%0D%0A%7D&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)
	var res = json2infoArray(data)

	if res == nil {
		return []info{info{"null", "null"}}
	}

	return res
}

func getClassName(queryResult []information) []string {

	var classNames = getPropResAll("http://www.w3.org/1999/02/22-rdf-syntax-ns#type", queryResult)

	// check if there is class information
	if classNames[0].typ.val == "null" {
		fmt.Println("no class information")
		return []string{"null", "null"}
	}

	//filter classnames
	var filter = [...]string{"http://dbpedia.org/ontology/Location", "http://dbpedia.org/ontology/Agent", "http://dbpedia.org/ontology/Person", "http://dbpedia.org/ontology/Place", "http://www.w3.org/2002/07/owl#Thing"}
	var keep = true

	// check dbo with filter
	for _, cn := range classNames {
		if cn.val.val == "null" {
			continue
		}
		if strings.Contains(cn.val.val, "dbpedia.org/ontology") {
			for _, f := range filter {

				if strings.Contains(cn.val.val, f) {
					keep = false
				}

			}
			if keep {
				fmt.Println("found dbo class matching filter " + cn.val.val)
				return []string{cn.val.val, cn.typ.val}

			}
		}
	}

	// check dbo
	for _, cn := range classNames {
		if cn.val.val == "null" {
			continue
		}
		if strings.Contains(cn.val.val, "dbpedia.org/ontology") {
			fmt.Println("found dbo class without filter " + cn.val.val)
			return []string{cn.val.val, cn.typ.val}
		}
	}

	var subClassOf = getPropResAll("http://www.w3.org/2000/01/rdf-schema#subClassOf", queryResult)
	keep = true

	// check subclassof with filter
	for _, cn := range subClassOf {
		if cn.val.val == "null" {
			continue
		}
		if strings.Contains(cn.val.val, "dbpedia.org/ontology") {
			for _, f := range filter {
				if strings.Contains(cn.val.val, f) {
					keep = false
				}

			}
			if keep {
				fmt.Println("found dbo subClassOf with filter " + cn.val.val)
				return []string{cn.val.val, cn.typ.val}
			}
		}
	}

	// check subclassof
	for _, cn := range subClassOf {
		if cn.val.val == "null" {
			continue
		}
		if strings.Contains(cn.val.val, "dbpedia.org/ontology") {
			fmt.Println("found dbo subClassOf without filter " + cn.val.val)
			return []string{cn.val.val, cn.typ.val}
		}
	}

	keep = true
	//check all with filter
	for _, cn := range classNames {
		if cn.val.val == "null" {
			continue
		}
		for _, f := range filter {

			if strings.Contains(cn.val.val, f) {
				keep = false
			}

		}
		if keep {
			fmt.Println("found any class with filter " + cn.val.val)
			return []string{cn.val.val, cn.typ.val}
		}
	}

	keep = true
	for _, cn := range subClassOf {

		if cn.val.val == "null" {
			continue
		}

		for _, f := range filter {

			if strings.Contains(cn.val.val, f) {
				keep = false
			}

		}
		if keep {
			fmt.Println("found any subClassOf with filter " + cn.val.val)
			return []string{cn.val.val, cn.typ.val}
		}
	}

	//fallback
	fmt.Println("found any class without filter " + classNames[0].val.val)
	return []string{classNames[0].val.val, classNames[0].typ.val}

	/*
		// determine the proper class of the item
		var classRDF = filterInfo(queryResult, []string{"rdf-syntax-ns#type", "rdf-schema#subClassOf"}, []string{"dbpedia.org/ontology", "owl#Class", "owl#Thing"})
		var classDBO = filterInfo(queryResult, []string{"dbpedia.org/ontology/type", "rdf-schema#subClassOf"}, []string{"dbpedia.org"})
		var class []information
		if len(classDBO) == 0 && len(classRDF) == 0 {

			//check for yago class
			// e.g. : http://dbpedia.org/class/yago/WikicatExtinctEarldomsInThePeerageOfEngland
			var classOther = filterInfo(queryResult, []string{"rdf-syntax-ns#type", "rdf-schema#subClassOf"}, []string{"dbpedia.org/class/"})

			if len(classOther) == 0 {
				//fmt.Println("Could not find class - setting class to Thing")
				//return []string{"Thing", "dbo"}
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

	*/
}

func getSiblings(class string) []info {

	// dbo
	fmt.Println("getting siblings for: " + class)
	class = cleanSpecialCharacters(class)
	var rq = "default-graph-uri=http://dbpedia.org&query=PREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0APREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0ASELECT+%3Fproperty%0D%0AWHERE+%7B+++++++%0D%0A++++++++%3Fproperty+dbo%3Atype+res%3A" + class + "+++%0D%0A%7D" + randlim + "&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
	var data = query(rq)
	var dbo = json2infoArray(data)

	//rdf
	rq = "default-graph-uri=http://dbpedia.org&query=PREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0APREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0APREFIX+rdf%3A+%3Chttp%3A%2F%2Fwww.w3.org%2F1999%2F02%2F22-rdf-syntax-ns%23%3E%0D%0ASELECT+%3Fproperty%0D%0AWHERE+%7B%0D%0A+++++++++%3Fproperty+rdf%3Atype+dbo%3A" + class + "+%0D%0A%7D" + randlim + "&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
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
	var rq = "default-graph-uri=http://dbpedia.org&query=PREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0APREFIX+rdfs%3A+%3Chttp%3A%2F%2Fwww.w3.org%2F2000%2F01%2Frdf-schema%23%3E%0D%0A%0D%0ASELECT+%3Fproperty+WHERE+%7B%0D%0A%09+++%09+%3Fproperty+rdfs%3AsubClassOf+dbo%3A" + superClass + "%0D%0A%09+++%7D" + randlim + "&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
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

func getAbstract(res []information) string {

	/*
		the	Article	1	1	Pre-primer	12
		be	Verb	2	2	Primer	21
		to	Preposition	3	7, 9	Pre-primer	17
		of	Preposition	4	4	Grade 1	12
		and	Conjunction	5	3	Pre-primer	16
		a	Article	6	5	Pre-primer	20
	*/

	// identify english abstract by selecting the one with the most occurences of common english words
	var eng = []string{"the", "to", "of", "and", "a", "The"}
	var results []information = make([]information, 0)

	for _, r := range res {
		if r.typ.val == "http://dbpedia.org/ontology/abstract" {
			results = append(results, r)
		}
	}

	if len(results) == 0 {
		fmt.Println("no abstract found")
		return "null"
	}

	var engAbstract = results[0]
	var maxScore = 0
	for _, abstract := range results {
		var score = 0

		for _, w := range eng {
			reg := regexp.MustCompile(" " + w + " ")
			matches := reg.FindAllStringIndex(abstract.val.val, -1)

			score += len(matches)

		}

		if score > maxScore {
			maxScore = score
			engAbstract = abstract
		}

	}

	return engAbstract.val.val
}

func getDepiction(res []information) string {
	var depiction = getPropRes("http://xmlns.com/foaf/0.1/depiction", res)
	if depiction.val.val == "null" {
		fmt.Println("no abstract found")

	}
	return depiction.val.val
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

func questionPropDist(d ItemData) Question {

	if d.propertyDists != nil && len(d.propertyDists) > 0 {

		var s = rand.NewSource(time.Now().Unix())
		var r = rand.New(s)
		propD := d.propertyDists[r.Intn(len(d.propertyDists))]

		var question = "The " + cleanURL(propD.property) + " of " + strings.ReplaceAll(d.item, "_", " ") + " is ..."
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

	var question = strings.ReplaceAll(d.item, "_", " ") + " is a ..."
	var answer = d.class

	var nD = len(d.siblingClasses)

	// check if siblings are present
	if d.siblingClasses != nil && len(d.siblingClasses) > 0 {

		// shuffle siblings
		rand.Seed(time.Now().UnixNano())

		dest := make([]info, len(d.siblingClasses))
		perm := rand.Perm(len(d.siblingClasses))
		for i, v := range perm {
			dest[v] = d.siblingClasses[i]
		}

		var ds []string = make([]string, 3)

		for idx, _ := range ds {

			if idx < nD {
				ds[idx] = cleanURL(dest[idx].val)
			} else {
				ds[idx] = ""
			}

		}

		var d1 = ds[0]
		var d2 = ds[1]
		var d3 = ds[2]
		//var d1 = cleanURL(dest[0].val)
		//var d2 = cleanURL(dest[1].val)
		//var d3 = cleanURL(dest[2].val)

		//clean Wikicat
		answer = strings.ReplaceAll(answer, "Wikicat", "")
		d1 = strings.ReplaceAll(d1, "Wikicat", "")
		d2 = strings.ReplaceAll(d2, "Wikicat", "")
		d3 = strings.ReplaceAll(d3, "Wikicat", "")

		return Question{question, answer, d1, d2, d3}
	} else {
		return getEmptyQuestion()
	}

}

func questionPropLiteral(d ItemData) Question {

	// new query if other fails:

	/*
			PREFIX res: <http://dbpedia.org/resource/>
			PREFIX dbo: <http://dbpedia.org/ontology/>
			PREFIX dbp: <http://dbpedia.org/property/>

			SELECT ?subject ?property WHERE {
		     	?subject dbp:population ?property
			}
			ORDER BY RAND()
			limit 10
	*/

	// select property

	if d.properties == nil || len(d.properties) == 0 {
		return getEmptyQuestion()
	} else {
		rand.Seed(time.Now().UnixNano())
		var idx = rand.Intn(len(d.properties))
		var p = d.properties[idx]

		// get distractors

		fmt.Println("getting property distractors for property " + p.typ.val)
		var rq = "default-graph-uri=http://dbpedia.org&query=%0D%0A%0D%0APREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0A%0D%0ASELECT++%3Fp+%3Fproperty+WHERE+%7B%0D%0A%09+++%09++%3Fp+a+dbo%3A" + d.superClass + "+%3B+%3C" + p.typ.val + "%3E%3Fproperty+.++%0D%0A%09+++%7D" + randlim + "&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"

		var data = query(rq)
		var res = json2infoArray(data)

		var t = 0
		if res == nil {

			for t < 5 && res == nil {

				fmt.Println("fallback to all objects ignoring superclass!!!")
				if t > 0 {
					rand.Seed(time.Now().UnixNano())
					idx = rand.Intn(len(d.properties))
					p = d.properties[idx]
				}

				rq = "default-graph-uri=http://dbpedia.org&query=PREFIX+res%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fresource%2F%3E%0D%0APREFIX+dbo%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fontology%2F%3E%0D%0APREFIX+dbp%3A+%3Chttp%3A%2F%2Fdbpedia.org%2Fproperty%2F%3E%0D%0A%0D%0ASELECT+%3Fproperty+WHERE+%7B%0D%0A+++++%3Fsubject+%3C" + strings.ReplaceAll(p.typ.val, "#", "%23") + "%3E%3Fproperty%0D%0A%7D" + randlim + "&format=application%2Fsparql-results%2Bjson&CXML_redir_for_subjs=121&CXML_redir_for_hrefs=&timeout=30000&debug=on&run=+Run+Query+"
				data = query(rq)
				res = json2infoArray(data)
				t += 1
			}
		}

		//if its still nill return empty question
		if res == nil {
			return getEmptyQuestion()
		}

		//var question = "The property " + cleanURL(p.typ.val) + " of " + strings.ReplaceAll(d.item, "_", " ") + " is ..."
		var question = "The " + cleanURL(p.typ.val) + " of " + strings.ReplaceAll(d.item, "_", " ") + " is ..."

		var answer = cleanURL(p.val.val)

		var tmp = make([]info, 0)
		for _, d := range res {

			var isInSet = false
			for _, i := range tmp {
				if i.val == d.val {
					isInSet = true
				}
			}

			if d.val != p.val.val && !isInSet {
				tmp = append(tmp, d)
			}
		}

		res = tmp

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

type QuestionWrapper struct {
	question Question
	img      string
	abstract string
}

func genQuestion(item string) QuestionWrapper {

	item = cleanStreetname(item)
	item = strings.TrimSpace(item)

	item = strings.ReplaceAll(item, " ", "_")

	fmt.Println("trying to generate a question for: " + item)

	// timeout

	var d = genItem(item, true, true)
	if d.item != "null" {
		item = d.item

	}

	var qs []Question = make([]Question, 1)
	qs[0] = rollQuestion(d)

	var tries = 0

	for tries < 10 && isEmptyQuestion(qs[0]) {
		qs[0] = rollQuestion(d)
		tries += 1
	}

	if isEmptyQuestion(qs[0]) {
		var disambi = d.disambiguations
		rand.Shuffle(len(disambi), func(i, j int) {
			disambi[i], disambi[j] = disambi[j], disambi[i]
		})

		for _, dis := range disambi {
			if dis == "null" {
				continue
			}

			d = genItem(dis, true, true)
			if d.item != "null" {
				item = d.item
			}

			qs[0] = rollQuestion(d)
			var tries = 0
			for tries < 10 && isEmptyQuestion(qs[0]) {
				qs[0] = rollQuestion(d)
				tries += 1
			}

			if !isEmptyQuestion(qs[0]) {
				qs[0] = cleanQuestion(qs[0])
				return QuestionWrapper{qs[0], d.depiction, d.abstract}
			}
		}

	} else {
		qs[0] = cleanQuestion(qs[0])
		return QuestionWrapper{qs[0], d.depiction, d.abstract}
	}

	// split words if _ inside
	if strings.Contains(item, "_") {
		words := strings.Split(item, "_")

		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(words), func(i, j int) { words[i], words[j] = words[j], words[i] })

		for _, w := range words {

			strings.ReplaceAll(w, ",", "")

			if strings.Contains(w, ")") || strings.Contains(w, "(") {
				continue
			}

			var isStopword = false
			for _, sw := range stopwordsItem {
				if w == sw || w == strings.Title(sw) {
					isStopword = true
				}
			}

			if !isStopword {
				return genQuestion(w)
			}
		}
	}

	if len(item) > 1 {
		item = item[:len(item)-1]
		return genQuestion(item)
	} else {
		var sryImage = "https://upload.wikimedia.org/wikipedia/commons/4/4e/Very_sorry.svg"
		return QuestionWrapper{Question{"What is 1+1?", "2", "3", "4", "42"}, sryImage, "Sorry no question could be generated."}
	}

	/*

		for isEmptyQuestion(qs[0]) && len(item) > 0 {

			rand.Seed(time.Now().UnixNano())

			if len(d.disambiguations) > 0 && d.disambiguations[0] != "null" {
				// pick random disambiguation
				rand.Shuffle(len(d.disambiguations), func(i, j int) {
					d.disambiguations[i], d.disambiguations[j] = d.disambiguations[j], d.disambiguations[i]
				})
				item = d.disambiguations[0]
			} else {
				//shorten string
				item = item[:len(item)-1]

			}

			d = genItem(item, true, true)

			//qs = []Question{rollQuestion(genItem(item, true, true))}
			//return []Question{getEmptyQuestion()}
		}

		//clean Question
		qs[0] = cleanQuestion(qs[0])

		return QuestionWrapper{qs[0], d.depiction, d.abstract}
	*/
}

func cleanQuestion(q Question) Question {

	q.answer = cleanURL(q.answer)
	q.d1 = cleanURL(q.d1)
	q.d2 = cleanURL(q.d2)
	q.d3 = cleanURL(q.d3)

	q.item = strings.ReplaceAll(q.item, "_", " ")
	q.answer = strings.ReplaceAll(q.answer, "_", " ")
	q.d1 = strings.ReplaceAll(q.d1, "_", " ")
	q.d2 = strings.ReplaceAll(q.d2, "_", " ")
	q.d3 = strings.ReplaceAll(q.d3, "_", " ")

	q.item = camelRegexp(q.item)
	q.answer = camelRegexp(q.answer)
	q.d1 = camelRegexp(q.d1)
	q.d2 = camelRegexp(q.d2)
	q.d3 = camelRegexp(q.d3)

	return q

}

func camelRegexp(str string) string {
	re := regexp.MustCompile(`([A-Z]+)`)
	str = re.ReplaceAllString(str, ` $1`)
	str = strings.Trim(str, " ")
	return str
}

func rollQuestion(d ItemData) Question {

	rand.Seed(time.Now().UnixNano())
	var roll = rand.Intn(101)

	if d.class != "null" && len(d.siblingClasses) > 0 {
		if roll < 40 {
			fmt.Println("generate SiblingClass Question")
			return questionSiblingClasses(d)
		} else if roll < 80 {
			fmt.Println("generate PropLiteral Question")
			return questionPropLiteral(d)
		} else {
			fmt.Println("generate PropDist Question")
			return questionPropDist(d)
		}
	} else {
		if roll < 70 {
			fmt.Println("generate PropLiteral Question")
			return questionPropLiteral(d)
		} else {
			fmt.Println("generate PropDist Question")
			return questionPropDist(d)
		}
	}

}

func isEmptyQuestion(q Question) bool {
	if q.item == "null" {
		return true
	} else {
		return false
	}
}

func getEmptyQuestion() Question {
	return Question{"null", "null", "null", "null", "null"}

}
