package main

import (
	"bytes"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

func databasePath(file string, environment string) string {
	yamlFile, err := ioutil.ReadFile(file)
	dbConfig := make(config)

	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(yamlFile, &dbConfig)
	if err != nil {
		panic(err)
	}

	buffer := new(bytes.Buffer)
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
		panic("socket or host must be set in database config")
	}

	buffer.WriteString("/" + database)

	return buffer.String()
}
