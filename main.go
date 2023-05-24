package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"text/template"
	"time"

	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/garyburd/go-oauth/oauth"
)

const name = "feed2twitter"

const version = "0.0.3"

var revision = "HEAD"

const (
	updateURL = "https://api.twitter.com/1.1/statuses/update.json"
)

var (
	oauthClient = oauth.Client{
		TemporaryCredentialRequestURI: "https://api.twitter.com/oauth/request_token",
		ResourceOwnerAuthorizationURI: "https://api.twitter.com/oauth/authenticate",
		TokenRequestURI:               "https://api.twitter.com/oauth/access_token",
	}
	dry        = flag.Bool("dry", false, "dry-run")
	silent     = flag.Bool("s", false, "no post")
	configFile = flag.String("c", "config.json", "path to config.json")
	issuesFile = flag.String("f", "issues.json", "path to issues.json")
)

type Feed2Twitter struct {
	bun.BaseModel `bun:"table:feed2twitter,alias:f"`

	Feed      string    `bun:"feed,pk,notnull" json:"feed"`
	GUID      string    `bun:"guid,pk,notnull" json:"guid"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

func postTweet(token *oauth.Credentials, status string) error {
	param := make(url.Values)
	param.Set("status", status)
	oauthClient.SignParam(token, "POST", updateURL, param)
	resp, err := http.PostForm(updateURL, url.Values(param))
	if err != nil {
		log.Println("failed to post tweet:", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("failed to post tweet")
		return err
	}
	return nil
}

func main() {
	var skip bool
	var dsn string
	var feedURL string
	var format string
	var clientToken, clientSecret, accessToken, accessSecret string
	var ver bool

	flag.BoolVar(&skip, "skip", false, "Skip tweet")
	flag.StringVar(&dsn, "dsn", os.Getenv("FEED2TWITTER_DSN"), "Database source")
	flag.StringVar(&feedURL, "feed", "", "Feed URL")
	flag.StringVar(&format, "format", "{{.Title}}\n{{.Link}}", "Tweet Format")
	flag.StringVar(&clientToken, "client-token", os.Getenv("FEED2TWITTER_CLIENT_TOKEN"), "Twitter ClientToken")
	flag.StringVar(&clientSecret, "client-secret", os.Getenv("FEED2TWITTER_CLIENT_SECRET"), "Twitter ClientSecret")
	flag.StringVar(&accessToken, "access-token", os.Getenv("FEED2TWITTER_ACCESS_TOKEN"), "Twitter AccessToken")
	flag.StringVar(&accessSecret, "access-secret", os.Getenv("FEED2TWITTER_ACCESS_SECRET"), "Twitter AccessSecret")
	flag.BoolVar(&ver, "v", false, "show version")
	flag.Parse()

	if ver {
		fmt.Println(version)
		os.Exit(0)
	}

	t := template.Must(template.New("").Parse(format))

	oauthClient.Credentials.Token = clientToken
	oauthClient.Credentials.Secret = clientSecret
	token := &oauth.Credentials{
		Token:  accessToken,
		Secret: accessSecret,
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	bundb := bun.NewDB(db, pgdialect.New())
	defer bundb.Close()

	_, err = bundb.NewCreateTable().Model((*Feed2Twitter)(nil)).IfNotExists().Exec(context.Background())
	if err != nil {
		log.Println(err)
		return
	}

	feed, err := gofeed.NewParser().ParseURL(feedURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for _, item := range feed.Items {
		if item == nil {
			break
		}

		fi := Feed2Twitter{
			Feed: feedURL,
			GUID: item.GUID,
		}
		_, err := bundb.NewInsert().Model(&fi).Exec(context.Background())
		if err != nil {
			log.Println(err)
			continue
		}

		var buf bytes.Buffer
		err = t.Execute(&buf, &item)
		if err != nil {
			log.Println(err)
			continue
		}
		content := buf.String()
		runes := []rune(content)
		if len(runes) > 140 {
			item.Title = string(item.Title[:len(item.Title)-len(runes)+140])
			buf.Reset()
			err = t.Execute(&buf, &item)
			if err != nil {
				log.Println(err)
				continue
			}
		}
		if skip {
			log.Printf("%q", buf.String())
			continue
		}
		err = postTweet(token, buf.String())
		if err != nil {
			log.Println(err)
			continue
		}
	}
}
