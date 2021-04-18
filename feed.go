package tinycast

import (
	"net/url"
	"path"
	"strings"

	"github.com/beevik/etree"
)

func modifyAttr(as []etree.Attr, attrName string, tr func(string) string) {
	for i, a := range as {
		if strings.ToLower(a.Key) == attrName {
			as[i].Value = tr(a.Value)
			continue
		}
	}
}

func removeAttr(as []etree.Attr, attrName string) []etree.Attr {
	j := 0
	for _, a := range as {
		if strings.ToLower(a.Key) != attrName {
			as[j] = a
			j++
		}
	}
	return as[:j]
}

func replace(doc *etree.Document, element string, baseUrl url.URL, qr url.Values) {
	for _, t := range doc.FindElements(element) {
		modifyAttr(t.Attr, "url", func(v string) string {
			bU := baseUrl
			query := qr
			query.Set("url", v)
			bU.RawQuery = query.Encode()
			bU.Path = path.Join(bU.Path, "convert.mp3")
			return bU.String()
		})
		t.Attr = removeAttr(t.Attr, "length")
		t.Attr = removeAttr(t.Attr, "filesize")
	}
}
