package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/dghubble/oauth1"
	twitter "github.com/g8rswimmer/go-twitter/v2"
)

const name = "feed2twitter"

const version = "0.0.12"

var revision = "HEAD"

type Feed2Twitter struct {
	bun.BaseModel `bun:"table:feed2twitter,alias:f"`

	Feed      string    `bun:"feed,pk,notnull" json:"feed"`
	GUID      string    `bun:"guid,pk,notnull" json:"guid"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

type authorize struct {
}

func (a authorize) Add(req *http.Request) {
}

func main() {
	var skip bool
	var dsn string
	var feedURL string
	var format string
	var pattern string
	var re *regexp.Regexp
	var clientToken, clientSecret, accessToken, accessSecret string
	var ver bool

	flag.BoolVar(&skip, "skip", false, "Skip tweet")
	flag.StringVar(&dsn, "dsn", os.Getenv("FEED2TWITTER_DSN"), "Database source")
	flag.StringVar(&feedURL, "feed", "", "Feed URL")
	flag.StringVar(&format, "format", "{{.Title | normalize}}\n{{.Link}}", "Tweet Format")
	flag.StringVar(&pattern, "pattern", "", "Match pattern")
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

	var err error
	if pattern != "" {
		re, err = regexp.Compile(pattern)
		if err != nil {
			log.Fatal(err)
		}
	}

	funcMap := template.FuncMap{
		"normalize": func(s string) string {
			// Remove invisible Unicode characters and squeeze multiple newlines
			s = regexp.MustCompile(`[\p{Cf}]`).ReplaceAllString(s, "")
			s = regexp.MustCompile(`\n\n+`).ReplaceAllString(s, "\n")
			return s
		},
	}
	t := template.Must(template.New("").Funcs(funcMap).Parse(format))

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

	config := oauth1.NewConfig(clientToken, clientSecret)
	client := &twitter.Client{
		Authorizer: authorize{},
		Client: config.Client(oauth1.NoContext, &oauth1.Token{
			Token:       accessToken,
			TokenSecret: accessSecret,
		}),
		Host: "https://api.twitter.com",
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
			if !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				log.Println(err)
			}
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

		content = buf.String()

		if re != nil {
			if !re.MatchString(content) {
				continue
			}
		}

		if skip {
			log.Printf("%q", content)
			continue
		}

		req := twitter.CreateTweetRequest{
			Text: content,
		}
		_, err = client.CreateTweet(context.Background(), req)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}
