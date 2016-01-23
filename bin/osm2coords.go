package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

var (
	ch chan bool
)

type Tag struct {
	Key   string `xml:"k"`
	Value string `xml:"v"`
}

type Node struct {
	Id       int     `xml:"id,attr"`
	Latitude float64 `xml:"lat,attr"`
	Longitde float64 `xml:"lon,attr"`
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

func main() {

	var node = flag.Bool("node", false, "")
	var way = flag.Bool("way", false, "")
	var rel = flag.Bool("rel", false, "")
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

	t1 := time.Now()

	if *node {
		n, _ := ProcessNode(*id)
		nodes = append(nodes, n)
	} else if *way {
		n, _ := ProcessWay(*id)
		nodes = n
	} else {
		n, _ := ProcessRel(*id)
		nodes = n
	}

	t2 := time.Since(t1)
	fmt.Printf("%d nodes in %v\n", len(nodes), t2)

	// sudo make me geojson...

	json.Marshal(nodes)
	// fmt.Printf("%s\n", str)

}
