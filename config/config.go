package config

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jinzhu/configor"
	"golib/http"
	"golib/log"
	"gopkg.in/yaml.v2"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ReadConfig(config any, uri string, ignoreError bool) error {
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	switch u.Scheme {
	case "env":
		return configor.Load(config)
	case "file":
		if err := readConfigFile(config, u); err != nil {
			if !ignoreError {
				return err
			}
			log.Warn("Failed to read config from file")
		}
		return configor.Load(config)
	case "http", "https":
		if err := readRemote(config, u); err != nil {
			if !ignoreError {
				return err
			}
			log.Warn("Failed to read config from file")
		}
		return configor.Load(config)
	default:
		return errors.New("unsupported scheme")
	}
}

func readConfigFile(config any, uri *url.URL) error {
	path := filepath.Join(uri.Host, uri.Path)
	ext := filepath.Ext(path)

	fileName := strings.TrimSuffix(path, ext) + ext

	switch ext {
	case ".json":
		jsonFile, err := os.ReadFile(fileName)
		if err != nil {
			return err
		}
		return json.Unmarshal(jsonFile, config)
	case ".yaml":
		yamlFile, err := os.ReadFile(fileName)
		if err != nil {
			return err
		}
		return yaml.Unmarshal(yamlFile, config)
	default:
		return errors.New("unsupported file format")
	}
}

func readRemote(config any, uri *url.URL) error {
	client := http.NewHttpClient(&http.Options{
		BaseUrl: uri.String(),
		Timeout: 10 * time.Second,
	})

	req := http.NewGETRequest("/", config, client)
	res, err := req.WithContext(context.Background()).
		IsRawResponse(true).
		Execute()
	if err != nil {
		return err
	}

	return json.Unmarshal(res.RawResponse, config)
}
