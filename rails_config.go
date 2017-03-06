package dnscheck

import (
	"bytes"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// A panicBuffer responds with a panic if a write to the underlying
// bytes.Buffer returns an error.
type panicBuffer struct {
	bytes.Buffer
}

func (pbuf *panicBuffer) ws(str ...string) {
	for _, s := range str {
		_, err := pbuf.WriteString(s)
		if err != nil {
			panic(err)
		}
	}
}

// RailsConfigToDSN tries to read a database config file typically found
// in a Ruby on Rails projects. These are structured first by an environment
// identifier (like "test", "development" or "production"), followed by
// the actual database config for that environment.
//
// Note, that each environment can have its own driver and setting pairs.
func RailsConfigToDSN(file string, environment string) (driver, dsn string) {
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	dbConfig := make(map[string]map[string]string)
	err = yaml.Unmarshal(yamlFile, &dbConfig)
	if err != nil {
		panic(err)
	}

	buffer := new(panicBuffer)
	envConfig := dbConfig[environment]

	if len(envConfig) == 0 {
		panic("invalid RAILS_ENV: " + environment)
	}

	host := envConfig["host"]
	port := envConfig["port"]
	socket := envConfig["socket"]
	username := envConfig["username"]
	password := envConfig["password"]
	database := envConfig["database"]

	buffer.ws(username)

	if password != "" {
		buffer.ws(":", password)
	}
	buffer.ws("@")

	if socket != "" {
		buffer.ws("unix(", socket, ")")
	} else if host != "" {
		if port == "" {
			port = "3306"
		}
		buffer.ws("tcp(", host, ":", port, ")")
	} else {
		panic("socket or host must be set in database config")
	}

	buffer.ws("/", database)
	dsn = buffer.String()

	switch envConfig["driver"] {
	case "mysql", "mysql2":
		driver = "mysql"
	default:
		panic("unsupported driver")
	}
	return
}
