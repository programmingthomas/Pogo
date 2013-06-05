package catcher

import (
	"encoding/xml"
)

type Fetched struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title string `xml:"title"`
	Link string `xml:"link"`
	Language string `xml:"langauge"`
	Copyright string `xml:"copyright"`
	Subtitle string `xml:"subtitle"`
	Author string `xml:"author"`
	Summary string `xml:"summary"`
	Description string `xml:"description"`
	Owner struct { 
		Name string `xml:"name"`
		Email string `xml:"email"`
	} `xml:"owner"`
	Image struct {
		Href string `xml:"href,attr"`
	} `xml:"image"`
	Items []Item `xml:"item"`
	Categories []Category `xml:"category"`
}

type Category struct {
	Text string `xml:"text,attr"`
	SubCategory struct {
		Text string `xml:"text,attr"`
	} `xml:"category"`
}

type Item struct {
	Title string `xml:"title"`
	Author string `xml:"author"`
	Subtitle string `xml:"subtitle"`
	Summary string `xml:"summary"`
	Description string `xml:"description"`
	Link string `xml:"guid"`
	Image struct {
		Href string `xml:"href,attr"`
	} `xml:"image"`
	PubDate string `xml:"pubDate"`
	Duration string `xml:"duration"`
	Enclosure struct {
		URL string `xml:"url,attr"`
		Length int64 `xml:"length,attr"`
		Type string `xml:"type,attr"`
	} `xml:"enclosure"`
}