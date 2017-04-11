package main

import (
	"encoding/json"
	"github.com/influxdata/influxdb/client/v2"
	"log"
	"os"
)

type db struct {
	Url      string
	Db       string
	Username string
	Password string
}

func (d *db) init() {
	//read config
	file, _ := os.Open("db_conf.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(d)
	checkError(err)
	log.Println("DB URL:", d.Url)
	log.Println("DB Name:", d.Db)
	log.Println("DB Username:", d.Username)
	log.Println("DB Password:", d.Password)
	file.Close()
}

func (d *db) queryInfo(id string) {
	var res []client.Result
	log.Println("ID:", id)
	log.Println("DB URL:", d.Url)
	// Create a new HTTPClient
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     d.Url,
		Username: d.Username,
		Password: d.Password,
	})
	checkError(err)
	q := client.Query{
		Command:  "drop series where uuid = '" + id + "'",
		Database: d.Db,
	}
	if response, err := c.Query(q); err == nil {
		if response.Error() != nil {
			log.Println("err1:", response.Error())
		}
		res = response.Results
	} else {
		log.Println("err2", err)
	}
	log.Println("Success")
	c.Close()
}
