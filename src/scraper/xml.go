package scraper

import "bytes"
import "code.google.com/p/go.net/html"
import "code.google.com/p/go.net/html/atom"

type parseArgs struct {
	atom     atom.Atom
	attr     string
	nodefunc func(value string) func(text string)
}

func ParseXml(xml []byte, atom atom.Atom, attr string,
	nodefunc func(value string) func(text string)) error {
	doc, err := html.Parse(bytes.NewReader(xml))
	if err != nil {
		return err
	}
	pa := parseArgs{atom, attr, nodefunc}
	pa.traverse(doc)
	return nil
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func (pa *parseArgs) getBytes(n *html.Node, buf *bytes.Buffer) {
	if n.Type == html.TextNode {
		buf.Write([]byte(n.Data))
		return
	}
	for cn := n.FirstChild; cn != nil; cn = cn.NextSibling {
		pa.getBytes(cn, buf)
	}
}

func (pa *parseArgs) traverse(n *html.Node) {
	if n.Type == html.ElementNode && n.DataAtom == pa.atom {
		value := getAttr(n, pa.attr)
		textfunc := pa.nodefunc(value)

		if textfunc != nil {
			var buf bytes.Buffer
			pa.getBytes(n, &buf)
			textfunc(string(buf.Bytes()))
			return
		}
	}
	for cn := n.FirstChild; cn != nil; cn = cn.NextSibling {
		pa.traverse(cn)
	}
}
