#!/usr/bin/env python

import sys
import os
import logging
import requests

try:
    import elementtree.ElementTree as ET
except Exception, e:
    import xml.etree.ElementTree as ET
        
def parse_rel(id):

    tree = fetch_el("relation", id)
    rel = tree.find('relation')

    for m in rel.findall('member'):

        type = m.attrib.get('type', None)
        ref = m.attrib.get('ref', None)

        if not ref:
            logging.error("failed to locate ref, skipping")
            continue

        if type == "way":
            for c in parse_way(ref):
                yield c
        elif type == "node":
            for c in parse_node(ref):
                yield c
        else:
            logging.error("unknown type (%s)" % type)
            continue

def parse_way(id):

    tree = fetch_el("way", id)
    way = tree.find("way")

    for n in way.findall("nd"):
        ref = n.attrib.get("ref", None)

        if not ref:
            logging.error("failed to locate ref, skipping")
            continue

        yield parse_node(ref)

def parse_node(id):

    tree = fetch_el("node", id)
    node = tree.find("node")

    lat = node.attrib.get("lat")
    lon = node.attrib.get("lon")

    return (lat, lon)

def fetch_el(el, id):

    url = "http://www.openstreetmap.org/api/0.6/%s/%s" % (el, id)
    logging.debug("fetch %s" % url)

    rsp = requests.get(url)

    tree = ET.fromstring(rsp.content)    
    return tree

if __name__ == '__main__':

    id = sys.argv[1]

    from polyline.codec import PolylineCodec
    pl = PolylineCodec().encode(list( parse_rel(id) ))

    print pl

    """
    for c in parse_rel(id):
        print c
    """
