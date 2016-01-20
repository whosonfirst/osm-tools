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

    try:
        tree = fetch_el("relation", id)
        rel = tree.find('relation')
    except Exception, e:
        logging.error("Failed to parse relation %s, because %s" % (id, e))
        yield None

    for m in rel.findall('member'):

        type = m.attrib.get('type', None)
        ref = m.attrib.get('ref', None)

        if not ref:
            logging.error("failed to locate ref, skipping")
            continue

        if type == "way":
            iter = parse_way(ref)
        elif type == "node":
            iter = parse_node(ref)
        else:
            logging.error("unknown type (%s)" % type)
            continue

        # this is wrong and broken but that it not today's problem...

        for i in iter:

            if not i:
                continue

            yield i.next()

def parse_way(id):

    try:
        tree = fetch_el("way", id)
        way = tree.find("way")
    except Exception, e:
        logging.error("Failed to parse way %s, because %s" % (id, e))
        yield None

    for n in way.findall("nd"):
        ref = n.attrib.get("ref", None)

        if not ref:
            logging.error("failed to locate ref, skipping")
            continue

        yield parse_node(ref)

def parse_node(id):

    try:
        tree = fetch_el("node", id)
        node = tree.find("node")
    except Exception, e:
        logging.error("Failed to parse node %s, because %s" % (id, e))
        yield None

    lat = node.attrib.get("lat")
    lon = node.attrib.get("lon")

    lat = float(lat)
    lon = float(lon)

    yield (lat, lon)

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
