package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	// "time"
)

var (
	ch chan bool
)

type Tag struct {
	Key   string `xml:"k"`
	Value string `xml:"v"`
}

type Node struct {
	Id        int     `xml:"id,attr"`
	Latitude  float64 `xml:"lat,attr"`
	Longitude float64 `xml:"lon,attr"`
	// Tags     []Tag   `xml:"tag"`
}

type Way struct {
	Id       int       `xml:"id,attr"`
	NodeRefs []NodeRef `xml:"nd"`
}

type NodeRef struct {
	Id int `xml:"ref,attr"`
}

type Rel struct {
	Id      int      `xml:"id,attr"`
	Members []Member `xml:"member"`
}

type Member struct {
	Type string `xml:"type,attr"`
	Id   int    `xml:"ref,attr"`
}

type GeoJSONFeature struct {
	Type       string            `json:"type"`
	Geometry   GeoJSONGeometry   `json:"geometry"`
	Properties GeoJSONProperties `json:"properties"`
	Id         int               `json:"id"`
}

type GeoJSONGeometry interface{}

type GeoJSONPoint struct {
	Type        string            `json:"type"`
	Coordinates GeoJSONCoordinate `json:"coordinates"`
}

type GeoJSONLineString struct {
	Type        string              `json:"type"`
	Coordinates []GeoJSONCoordinate `json:"coordinates"`
}

type GeoJSONProperties struct {
	Type string `json:"type"`
}

type GeoJSONCoordinate []float64

func (c GeoJSONCoordinate) Latitude() float64 {
	return c[1]
}

func (c GeoJSONCoordinate) Longitude() float64 {
	return c[0]
}

func ProcessRel(id int) ([]*Node, error) {

	rsp, err := Fetch("relation", id)

	if err != nil {
		return nil, err
	}

	// fmt.Printf("%s", rsp)

	type OSM struct {
		XMLName xml.Name `xml:"osm"`
		Rel     Rel      `xml:"relation"`
	}

	osm := OSM{}
	err = xml.Unmarshal(rsp, &osm)

	if err != nil {
		return nil, err
	}

	count := len(osm.Rel.Members)
	tmp := make([][]*Node, count)

	wg := new(sync.WaitGroup)

	for idx, m := range osm.Rel.Members {

		wg.Add(1)

		go func(idx int, m Member) {

			defer wg.Done()

			if m.Type == "relation" {
				nodes, err := ProcessRel(m.Id)

				if err != nil {
					fmt.Println(err)
				}

				tmp[idx] = nodes

			} else if m.Type == "way" {
				nodes, err := ProcessWay(m.Id)

				if err != nil {
					fmt.Println(err)
				}

				tmp[idx] = nodes
			} else if m.Type == "node" {
				node, err := ProcessNode(m.Id)

				if err != nil {
					fmt.Println(err)
				}

				tmp[idx] = []*Node{node}
			} else {
				fmt.Printf("%d is unknown\n", idx)
			}

		}(idx, m)
	}

	wg.Wait()

	nodes := make([]*Node, 0)

	for _, _nodes := range tmp {

		for _, _node := range _nodes {
			nodes = append(nodes, _node)
		}
	}

	return nodes, nil
}

func ProcessWay(id int) ([]*Node, error) {

	rsp, err := Fetch("way", id)

	if err != nil {
		return nil, err
	}

	// fmt.Printf("%s", rsp)

	type OSM struct {
		XMLName xml.Name `xml:"osm"`
		Way     Way      `xml:"way"`
	}

	osm := OSM{}
	err = xml.Unmarshal(rsp, &osm)

	if err != nil {
		return nil, err
	}

	count := len(osm.Way.NodeRefs)
	nodes := make([]*Node, count)

	wg := new(sync.WaitGroup)

	for idx, n := range osm.Way.NodeRefs {

		wg.Add(1)

		go func(idx int, n NodeRef) {

			defer wg.Done()

			node, err := ProcessNode(n.Id)

			if err != nil {
				return
			}

			nodes[idx] = node
		}(idx, n)
	}

	wg.Wait()

	return nodes, nil
}

func ProcessNode(id int) (*Node, error) {

	rsp, err := Fetch("node", id)

	if err != nil {
		return nil, err
	}

	type OSM struct {
		XMLName xml.Name `xml:"osm"`
		Node    Node     `xml:"node"`
	}

	osm := OSM{}
	err = xml.Unmarshal(rsp, &osm)

	if err != nil {
		return nil, err
	}

	node := osm.Node
	return &node, nil
}

func Fetch(el string, id int) ([]byte, error) {

	<-ch

	defer func() { ch <- true }()

	url := fmt.Sprintf("http://www.openstreetmap.org/api/0.6/%s/%d", el, id)
	// fmt.Println(url)

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}

func Nodes2GeoJSON(el string, id *int, nodes []*Node) GeoJSONFeature {

	var geom GeoJSONGeometry

	if el == "node" {

		coords := GeoJSONCoordinate{nodes[0].Longitude, nodes[0].Latitude}
		geom = GeoJSONPoint{"Point", coords}

	} else if el == "way" {

		coords := make([]GeoJSONCoordinate, 0)

		for _, node := range nodes {
			c := GeoJSONCoordinate{node.Longitude, node.Latitude}
			coords = append(coords, c)
		}

		geom = GeoJSONLineString{"LineString", coords}

	} else {

		// Please write me...
		// https://wiki.openstreetmap.org/wiki/Relation
		// https://wiki.openstreetmap.org/wiki/Types_of_relation
	}

	props := GeoJSONProperties{
		Type: el,
	}

	feature := GeoJSONFeature{"Feature", geom, props, *id}

	return feature
}

func main() {

	var node = flag.Bool("node", false, "")
	var way = flag.Bool("way", false, "")
	var rel = flag.Bool("rel", false, "")
	var geojson = flag.Bool("geojson", false, "")
	var id = flag.Int("id", 0, "")
	var procs = flag.Int("procs", 100, "")

	flag.Parse()

	if !*node && !*way && !*rel {
		panic("SAD")
	}

	if *id == 0 {
		panic("ZERO")
	}

	ch = make(chan bool, *procs)

	for i := 0; i < *procs; i++ {
		go func() { ch <- true }()
	}

	nodes := make([]*Node, 0)

	// node: 3668644956
	// way: 169202638
	// rel: 2128634

	// t1 := time.Now()

	if *rel && *geojson {
		fmt.Println("GeoJSON exports for relations are not supported yet. Sad face.")
		os.Exit(1)
	}

	var el string

	if *node {
		el = "node"
		n, _ := ProcessNode(*id)
		nodes = append(nodes, n)
	} else if *way {
		el = "way"
		n, _ := ProcessWay(*id)
		nodes = n
	} else {
		el = "rel"
		n, _ := ProcessRel(*id)
		nodes = n
	}

	// t2 := time.Since(t1)
	// fmt.Printf("%d nodes in %v\n", len(nodes), t2)

	var export interface{}
	export = nodes

	if *geojson {
		export = Nodes2GeoJSON(el, id, nodes)
	}

	str, _ := json.Marshal(export)
	fmt.Printf("%s\n", str)

	os.Exit(0)
}
