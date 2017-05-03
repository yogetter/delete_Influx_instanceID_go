package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type openstackConf struct {
	Tenantname  string
	Username    string
	Password    string
	OS_AUTH_URL string
}

func (o *openstackConf) init() {
	data, _ := os.Open("openstack_conf.json")
	decoder := json.NewDecoder(data)
	err := decoder.Decode(o)
	checkError(err)
}

func (o *openstackConf) getTokenUrl(json_data []byte) (string, string) {
	var tmp interface{}
	client := &http.Client{}
	req, err := http.NewRequest("POST", o.OS_AUTH_URL+"/tokens", bytes.NewBuffer(json_data))
	checkError(err)

	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)

	defer res.Body.Close()
	o.ioRead(res, &tmp)

	res_data := tmp.(map[string]interface{})["access"].(map[string]interface{})
	url := res_data["serviceCatalog"].([]interface{})[0].(map[string]interface{})["endpoints"].([]interface{})[0].(map[string]interface{})["adminURL"]
	token := res_data["token"].(map[string]interface{})["id"]
	return token.(string), url.(string)
}
func (o *openstackConf) ioRead(r *http.Response, f *interface{}) {
	body, err := ioutil.ReadAll(r.Body)
	dec := json.NewDecoder(strings.NewReader(string(body)))
	err = dec.Decode(f)
	checkError(err)

}
func (o *openstackConf) insertInstance(res_data []interface{}) []string {
	var tmp []string
	for _, value := range res_data {
		tmp = append(tmp, value.(map[string]interface{})["id"].(string))
	}
	return tmp
}
func (o *openstackConf) getInstances(influx *db) []interface{} {
	var json_data = []byte(`{"auth":{"tenantName":"` + o.Tenantname + `","passwordCredentials":{"username":"` + o.Username + `", "password":"` + o.Password + `"}}}`)
	var tmp interface{}
	token, url := o.getTokenUrl(json_data)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url+"/servers", nil)
	checkError(err)
	req.Header.Set("X-Auth-Token", token)
	res, err := client.Do(req)
	defer res.Body.Close()
	o.ioRead(res, &tmp)

	res_data := tmp.(map[string]interface{})["servers"].([]interface{})
	return res_data
}

func deleteData(influx *db) {
	// Found ID should be delete
	flag := true
	for _, data1 := range db_instances {
		for _, data2 := range live_instances {
			if data1[1] == data2 {
				flag = false
				log.Println("same:", data1)
				break
			}
		}
		if flag {
			log.Println("Delete id:", data1)
			log.Println("influx Url:", influx.Url)
			influx.queryInfo("'"+data1[1].(string)+"'", "drop series where uuid = ")
		}
		flag = true
	}
}
func queryData(influx *db) {
	tmp_data := influx.queryInfo("uuid", "show tag values from vm_usage with key = ")
	if tmp_data[0].Series != nil {
		//log.Println("Tmp_data:", tmp_data[0], "value:", tmp_data[0].Series)
		db_instances = tmp_data[0].Series[0].Values
	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var run openstackConf
var db_instances [][]interface{}
var live_instances []string
var influx db

func main() {
	influx := db{}
	influx.init()
	run = openstackConf{}
	run.init()
	for {
		rep_data := run.getInstances(&influx)
		live_instances = run.insertInstance(rep_data)
		queryData(&influx)
		log.Println("InDb:", db_instances)
		log.Println("Live:", live_instances)
		deleteData(&influx)
		time.Sleep(60 * time.Second)
	}
}
