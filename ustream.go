package ustream

import (
  "log"
  "net/http"
  "bufio"
  "io/ioutil"
  "bytes"
  "strings"
  "github.com/kurrik/oauth1a"
  "encoding/json"
)

type Tweet struct {
  Id_str string
  Text string
  User *User
  In_reply_to_status_id_str string
}

type User struct {
  Screen_name string
  Name string
}

type UStreamClient struct {
  service *oauth1a.Service
  user *oauth1a.UserConfig
  httpClient *http.Client
  stream chan *Tweet
}

func NewUStreamClient() *UStreamClient {

  creds, err := ReadCredentials()
  if err != nil {
    log.Printf("Credential read error, could not create UStreamClient")
    return nil
  }

  u := new(UStreamClient)

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

func (u *UStreamClient) Connect() (*http.Response, error) {

  httpRequest, _ := http.NewRequest("GET", "https://userstream.twitter.com/1.1/user.json", nil)
  u.service.Sign(httpRequest, u.user)
  var httpResponse *http.Response
  var err error
  httpResponse, err = u.httpClient.Do(httpRequest)

  if err != nil {
    log.Printf("Connection error")
    return nil, err
  }

  return httpResponse, nil
}

func (u *UStreamClient) ReadStream(resp *http.Response) (chan *Tweet) {
  var reader *bufio.Reader
  reader = bufio.NewReader(resp.Body)

  go func() {
  for {

      line, err := reader.ReadBytes('\n')

      if err != nil {
        log.Printf("Error reading line of response\n")
      }

      line = bytes.TrimSpace(line)

      if len(line) == 0 {
          continue
      }

      // Skip over friendlist and events for now
      if bytes.HasPrefix(line, []byte(`{"event":`)) {
        continue
      }
      if bytes.HasPrefix(line, []byte(`{"friends":`)) {
        continue
      }

      // Unmarshal the tweet into a buffer
      var buffer map[string]interface{}
      json.Unmarshal(line, &buffer)

      var tweet *Tweet

      // Only grab the information we want from the unmarshalled data
      if buffer["id"] != 0 && buffer["text"] != "" {

        id_str, ok := buffer["id_str"].(string)

        if !ok {
          log.Printf("Error converting Tweet ID: %#v", buffer["id_str"])
          continue
        }

        text, ok := buffer["text"].(string)

        if !ok {
          log.Printf("Error converting Tweet text")
          continue
        }

        in_reply_to_status_id_str, ok := buffer["in_reply_to_status_id_str"].(string)

        if string(in_reply_to_status_id_str) == "null" {
          log.Printf("Error converting Tweet reply status id str")
        }

        if buffer["user"] != nil {

          user, ok := buffer["user"].(map[string]interface{})

          if !ok {
            log.Printf("Error converting Tweet User")
            continue
          }

          name, ok := user["name"].(string)

          if !ok {
            log.Printf("Error converting Tweet User Name")
            continue
          }

          screen_name, ok := user["screen_name"].(string)

          if !ok {
            log.Printf("Error converting Tweet User ScreenName")
            continue
          }

          // Create Tweet
          tweet = &Tweet{ id_str, text, &User{ screen_name, name }, in_reply_to_status_id_str}
        }
      }

      u.stream <- tweet
  }
  }()

  return u.stream
}
