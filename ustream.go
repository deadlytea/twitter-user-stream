package main

import (
  "fmt"
  "net/http"
  "bufio"
  "io/ioutil"
  "bytes"
  "strings"
  "github.com/kurrik/oauth1a"
  // "json"
)

type Tweet struct {
  id int64
  text string
}

// TODO: rename this to a client
type UStreamConn struct {
  service *oauth1a.Service
  user *oauth1a.UserConfig
  httpClient *http.Client
  stream chan *Tweet
}

func NewUStreamConn() *UStreamConn {

  creds, err := ReadCredentials()
  if err != nil {
    fmt.Printf("Credential read error, could not create UstreamConnection")
    return nil
  }

  u := new(UStreamConn)

  u.service = &oauth1a.Service{
    RequestURL:   "https://api.twitter.com/oauth/request_token",
    AuthorizeURL: "https://api.twitter.com/oauth/request_token",
    AccessURL:    "https://api.twitter.com/oauth/request_token",
    ClientConfig: &oauth1a.ClientConfig{
        ConsumerKey:    creds["oauth_consumer_key"],
        ConsumerSecret: creds["oauth_consumer_secret"],
        CallbackURL:    "http://www.deadlytea.com", // unused
    },
    Signer: new(oauth1a.HmacSha1Signer),
  }

  u.httpClient = &http.Client{
    Transport: &http.Transport{},
  }

  u.user = oauth1a.NewAuthorizedConfig(creds["oauth_token"], creds["oauth_token_secret"])

  u.stream = make(chan *Tweet, 10)

  return u
}

func ReadCredentials() (map[string]string, error) {
  creds, err := ioutil.ReadFile("CREDENTIALS")
  if err != nil {
    return nil, err
  }

  lines := strings.Split(string(creds), "\n")

  c := make(map[string]string)

  c["oauth_consumer_key"] = lines[0]
  c["oauth_consumer_secret"] = lines[1]
  c["oauth_token"] = lines[2]
  c["oauth_token_secret"] = lines[3]

  return c, nil
}

func (u *UStreamConn) Connect() (error) {

  httpRequest, _ := http.NewRequest("GET", "https://userstream.twitter.com/1.1/user.json", nil)
  u.service.Sign(httpRequest, u.user)
  var httpResponse *http.Response
  var err error
  httpResponse, err = u.httpClient.Do(httpRequest)

  if err != nil {
    fmt.Printf("Connection error")
    return err
  }

  u.ReadStream(httpResponse)

  return nil
}

func (u *UStreamConn) ReadStream(resp *http.Response) {
  var reader *bufio.Reader
  reader = bufio.NewReader(resp.Body)
  for {

      line, err := reader.ReadBytes('\n')

      if err != nil {
        fmt.Printf("Error reading line of response")
      }

      line = bytes.TrimSpace(line)

      if len(line) == 0 {
          continue
      }

      fmt.Printf(string(line[:]))

  }
}

func main() {
  client := NewUStreamConn()

  client.Connect()
}
