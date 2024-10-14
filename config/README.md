# Config Reader

Read Configuration from env, file, or from url

## Hierarchy

Supported schema:
- env
    - json
    - yaml
    - http/https


Config will always load from env. For example, if you are loading config from file, then it can be replaced by env variables if it exists.

## Usage

For example we have this configuration:

```go
type (
	TestConfig struct {
		TestVal string   `env:"TEST_VAL" json:"test_val" yaml:"TestVal"`
		DbConf  DbConfig `yaml:"DbConf" json:"db_conf"`
	}
	DbConfig struct {
		Host string `yaml:"Host" json:"host" env:"DB_HOST"`
		Port int    `yaml:"Port" json:"port" env:"DB_PORT"`
	}
)
```

Defining tag will read configuration from it. If you only want to get it from json/yaml you can add tag `json` or `yaml` only. If you add tag `env` then it can be replaceable by env.

### Read Only From Env

```go
func main() {
	var cfg TestConfig

	err := config.ReadConfig(&cfg, "env://", false)
	if err != nil {
		log.Fatal("failed to read env")
	}
}
```

### Read From Json File and Env
```go
func main() {
	var cfg TestConfig

	err := config.ReadConfig(&cfg, "file://{path}/config.json", false)
	if err != nil {
		log.Fatal("failed to read json config file")
	}
}
```
You need to defined path and config file. If your config struct have any tag `env` then it can be replaceable by env.


### Read From Yaml File and Env
```go
func main() {
	var cfg TestConfig

	err := config.ReadConfig(&cfg, "file://{path}/config.yaml", false)
	if err != nil {
		log.Fatal("failed to read yaml config file")
	}
}
```
You need to defined path and config file. If your config struct have any tag `env` then it can be replaceable by env.


### Read From Url and Env
You must ensure that url returning json not file.
```go
func main() {
	var cfg TestConfig

	err := config.ReadConfig(&cfg, "http://localhost:8900/v1/config", false)
	if err != nil {
		log.Fatal("failed to read from url")
	}
}
```