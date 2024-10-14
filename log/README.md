# Log

This log utilizing logrus from `github.com/sirupsen/logrus`. By default, this library will log caller file and line to make debugging easier.

## Usage

By default, log will use level `info` and text formatter. If you want to change it, you can set it on top level call.

Sample usage:
```go
package main

import "git-rbi.jatismobile.com/bento-library/go-lib/code-base-lib/log"

func main() {
	log.SetLevel("info")
	log.SetFormatter("json")
	log.Info("log here")
}

```

Sample log output:
```
{"level":"info","message":"log here","source":"main.go:8","time":"2023-11-03T22:33:41+07:00","request-id":""}
```