package main

import (
	"bytes"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

func databasePath(file string, environment string) string {
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("cannot read %q: %v", file, err)
	}

	dbConfig := make(map[string]map[string]string)

	err = yaml.Unmarshal(yamlFile, &dbConfig)
	if err != nil {
		log.Fatalf("cannot parse %q: %v", file, err)
	}

	buffer := new(bytes.Buffer)
	envConfig := dbConfig[environment]

	if len(envConfig) == 0 {
		log.Fatalf("invalid RAILS_ENV: %q", environment)
	}

	host := envConfig["host"]
	port := envConfig["port"]
	socket := envConfig["socket"]
	username := envConfig["username"]
	password := envConfig["password"]
	database := envConfig["database"]

	buffer.WriteString(username)

	if password != "" {
		buffer.WriteString(":" + password)
	}
	buffer.WriteString("@")

	if socket != "" {
		buffer.WriteString("unix(" + socket + ")")
	} else if host != "" {
		buffer.WriteString("tcp(" + host + ":")
		if port == "" {
			buffer.WriteString("3306")
		} else {
			buffer.WriteString(port)
		}
		buffer.WriteString(")")
	} else {
		log.Fatal("socket or host must be set in database config")
	}

	buffer.WriteString("/" + database)

	return buffer.String()
}
