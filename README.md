# feed2twitter

Post tweet via RSS feed.

## Usage

```
feed2twitter -dsn $DATABASE_URL \
    -feed https://vim-jp.org/rss.xml \
    -format '{{.Title}}{{"\n"}}{{.Link}} #vimeditor'
```

Or kubernetes cronjob.

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: vim-jp-feed-bot
spec:
  schedule: '0 * * * *'
  successfulJobsHistoryLimit: 1
  failedJobsHistoryLimit: 1
  jobTemplate:
    spec:
      backoffLimit: 1
      template:
        spec:
          containers:
          - name: vim-jp-feed-bot
            image: mattn/feed2witter
            imagePullPolicy: IfNotPresent
            #imagePullPolicy: Always
            command: ["/go/bin/feed2twitter"]
            args:
            - '-dsn'
            - '-feed'
            - 'https://vim-jp.org/rss.xml'
            - '-format'
            - '{{.Title}}{{\"\n\"}}{{.Link}} #vimeditor'
            env:
            - name: FEED2TWITTER_CLIENT_TOKEN
              valueFrom:
                configMapKeyRef:
                  name: vim-jp-feed-bot
                  key: feed2twitter-client-token
            - name: FEED2TWITTER_CLIENT_SECRET
              valueFrom:
                configMapKeyRef:
                  name: vim-jp-feed-bot
                  key: feed2twitter-client-secret
            - name: FEED2TWITTER_ACCESS_TOKEN
              valueFrom:
                configMapKeyRef:
                  name: vim-jp-feed-bot
                  key: feed2twitter-access-token
            - name: FEED2TWITTER_ACCESS_SECRET
              valueFrom:
                configMapKeyRef:
                  name: vim-jp-feed-bot
                  key: feed2twitter-access-secret
            - name: FEED2TWITTER_DSN
              valueFrom:
                configMapKeyRef:
                  name: vim-jp-feed-bot
                  key: feed2twitter-dsn
          restartPolicy: Never
```

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vim-jp-feed-bot
data:
  feed2twitter-client-token: 'XXXXXXXXXXXXXXXXXXXXXX'
  feed2twitter-client-secret: 'XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX'
  feed2twitter-access-token: 'XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX'
  feed2twitter-access-secret: 'XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX'
  feed2twitter-dsn: 'postgres://user:password@server/database'
```

## Installation

```
$ go install github.com/mattn/feed2twitter@latest
```

Or use `mattn/feed2twitter` for docker image.

## License

MIT

## Author

Yasuihro Matsumoto (a.k.a. mattn)
