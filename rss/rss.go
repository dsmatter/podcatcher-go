package rss

import (
  "encoding/xml"
  "io/ioutil"
  "path"
  "net/http"
  "strings"
)

type Item struct {
  Title string `xml:"title"`
  Enclosure struct {
    URL string `xml:"url,attr"`
  } `xml:"enclosure"`
}

type Channel struct {
  Title string `xml:"title"`
  Items []Item `xml:"item"`
}

func FeedLinks(dir string) ([]string, error) {
  feedUrlPath := path.Join(dir, "feed.url")
  feedUrlBuffer, err := ioutil.ReadFile(feedUrlPath)
  if err != nil {
    return nil, err
  }
  feedUrl := strings.TrimSpace(string(feedUrlBuffer))

  res, err := http.Get(feedUrl)
  if err != nil {
    return nil, err
  }

  defer res.Body.Close()
  xmlDecoder := xml.NewDecoder(res.Body)
  var rss struct {
    Channel Channel `xml:"channel"`
  }
  err = xmlDecoder.Decode(&rss)
  if err != nil {
    return nil, err
  }
  result := make([]string, len(rss.Channel.Items))
  for i, item := range rss.Channel.Items {
    result[i] = item.Enclosure.URL
  }
  return result, nil
}
