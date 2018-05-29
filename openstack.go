package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type openstackConf struct {
	OS_AUTH_URL string
	NOVA_ENDPOINT string
}

func (o *openstackConf) Init() {
	data, _ := os.Open("openstack_conf.json")
	decoder := json.NewDecoder(data)
	err := decoder.Decode(o)
	CheckError(err)
}

func (o *openstackConf) GetUrl(catalog interface{}) {
	for _, value := range catalog.([]interface{}){
		if value.(map[string]interface{})["name"] == "nova" {
			for _, url := range value.(map[string]interface{})["endpoints"].([]interface{}) {
				if url.(map[string]interface{})["interface"] == "internal" {
					o.NOVA_ENDPOINT = url.(map[string]interface{})["url"].(string)
				}
			}
		}
	}
}

func (o *openstackConf) GetToken() (string) {
	var ResponseData interface{}
	data, _ := os.Open("user_info.json")
	client := &http.Client{}
	req, err := http.NewRequest("POST", o.OS_AUTH_URL+":5000/v3/auth/tokens", data)
	CheckError(err)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	defer res.Body.Close()
	o.IoRead(res, &ResponseData)
	catalog := ResponseData.(map[string]interface{})["token"].(map[string]interface{})["catalog"]
	o.GetUrl(catalog)
	token := res.Header.Get("X-Subject-Token")
	return token
}
func (o *openstackConf) IoRead(r *http.Response, f *interface{}) {
	body, err := ioutil.ReadAll(r.Body)
	dec := json.NewDecoder(strings.NewReader(string(body)))
	err = dec.Decode(f)
	CheckError(err)

}
func (o *openstackConf) InsertInstance(res_data []interface{}) []string {
	var tmp []string
	for _, value := range res_data {
		tmp = append(tmp, value.(map[string]interface{})["id"].(string))
	}
	return tmp
}
func (o *openstackConf) GetInstances(influx *db) []interface{} {
	var tmp interface{}
	token := o.GetToken()
	client := &http.Client{}
	req, err := http.NewRequest("GET", o.NOVA_ENDPOINT + "/servers?all_tenants", nil)
	CheckError(err)
	req.Header.Set("X-Auth-Token", token)
	res, err := client.Do(req)
	defer res.Body.Close()
	o.IoRead(res, &tmp)
	log.Println(tmp)
	res_data := tmp.(map[string]interface{})["servers"].([]interface{})
	log.Println(res_data)
	return res_data
}

func DeleteData(influx *db) {
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
func QueryData(influx *db) {
	tmp_data := influx.queryInfo("uuid", "show tag values from vm_usage with key = ")
	if tmp_data[0].Series != nil {
		//log.Println("Tmp_data:", tmp_data[0], "value:", tmp_data[0].Series)
		db_instances = tmp_data[0].Series[0].Values
	}
}

func CheckError(err error) {
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
	influx.Init()
	run = openstackConf{}
	run.Init()
	for {
		rep_data := run.GetInstances(&influx)
		live_instances = run.InsertInstance(rep_data)
		QueryData(&influx)
		log.Println("InDb:", db_instances)
		log.Println("Live:", live_instances)
		DeleteData(&influx)
		time.Sleep(60 * time.Second)
	}
}
