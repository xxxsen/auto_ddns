package main

import (
	"flag"
	"fmt"
	"encoding/xml"
	"time"
	"net/http"
	"io/ioutil"
)

var URL_GET_DNS_DATA = "https://www.namesilo.com/api/dnsListRecords?version=1&type=xml&key=%s&domain=%s"
var URL_UPDATE_DNS_DATA = "https://www.namesilo.com/api/dnsUpdateRecord?version=1" +
	"&type=xml&key=%s&domain=%s&rrid=%s&rrhost=%s&rrvalue=%s&rrttl=%d"

var domain = flag.String("domain", "example.com", "your main domain")
var subDomain = flag.String("sub", "home", "sub domain")
var key = flag.String("api_key", "xxx", "api key")
var ttl = flag.Int("ttl", 3600, "ttl")
var circle = flag.Int("circle", 300, "circle")

type DnsData struct {
	OutterIP  string
	RecordId  string
	CurrentIP string
}

type ListDnsRequest struct {
	Operation string `xml:"operation"`
	IP        string `xml:"ip"`
}

type ResourceRecordData struct {
	RecordId string `xml:"record_id"`
	Type     string `xml:"type"`
	Host     string `xml:"host"`
	Value    string `xml:"value"`
	TTL      int    `xml:"ttl"`
	Distance int    `xml:"distance"`
}

type ListReply struct {
	Code           int                  `xml:"code"`
	Detail         string               `xml:"detail"`
	ResourceRecord []ResourceRecordData `xml:"resource_record"`
}

type ListDnsDataParse struct {
	Namesilo xml.Name       `xml:"namesilo"`
	Request  ListDnsRequest `xml:"request"`
	Reply    ListReply      `xml:"reply"`
}

type UpdateRequest struct {
	Operation string `xml:"operation"`
	IP        string `xml:"ip"`
}

type UpdateReply struct {
	Code     int    `xml:"code"`
	Detail   string `xml:"detail"`
	RecordId string `xml:"record_id"`
}

type UpdateDataParse struct {
	Namesilo xml.Name      `xml:"namesilo"`
	Request  UpdateRequest `xml:"request"`
	Reply    UpdateReply   `xml:"reply"`
}

func RequestHTTP(url string) ([]byte, error) {
	rsp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("Get data from remote:%s", string(data))
	return data, nil
}

func GetDNSData(domain string, sub string, key string) (*DnsData, error) {
	url := fmt.Sprintf(URL_GET_DNS_DATA, key, domain)
	data, err := RequestHTTP(url)
	if err != nil {
		return nil, err
	}
	parser := &ListDnsDataParse{}
	if err = xml.Unmarshal(data, parser); err != nil {
		return nil, err
	}
	if parser.Reply.Code != 300 {
		return nil, fmt.Errorf("request err, code:%d, msg:%s", parser.Reply.Code, parser.Reply.Detail)
	}
	target := fmt.Sprintf("%s.%s", sub, domain)
	var dnsData *DnsData = nil
	for _, item := range parser.Reply.ResourceRecord {
		if item.Host == target {
			dnsData = &DnsData{}
			dnsData.OutterIP = parser.Request.IP
			dnsData.RecordId = item.RecordId
			dnsData.CurrentIP = item.Value
		}
	}
	//fmt.Printf("%+v", *parser)
	if dnsData == nil {
		return nil, fmt.Errorf("not found target host:%s", target)
	}
	return dnsData, nil
}

func UpdateDNSData(domain, sub, recordid string, key string, value string, ttl int) error {
	url := fmt.Sprintf(URL_UPDATE_DNS_DATA, key, domain, recordid, sub, value, ttl)
	data, err := RequestHTTP(url)
	if err != nil {
		return err
	}
	parser := &UpdateDataParse{}
	if err = xml.Unmarshal(data, parser); err != nil {
		return err
	}
	if parser.Reply.Code != 300 {
		return fmt.Errorf("code not ok, code:%d, msg:%s", parser.Reply.Code, parser.Reply.Detail)
	}
	return nil
}

func doCircle() {
	for {
		for {
			data, err := GetDNSData(*domain, *subDomain, *key)
			if err != nil {
				fmt.Printf("Get dns data fail, err:%v, domain:%s, sub:%s\n", err, *domain, *subDomain)
				break
			}
			fmt.Printf("Read dns data from namesilo succ, recordid:%s, current ip:%s, outter ip:%s\n",
				data.RecordId, data.CurrentIP, data.OutterIP)
			err = UpdateDNSData(*domain, *subDomain, data.RecordId, *key, data.OutterIP, *ttl)
			if err != nil {
				fmt.Printf("Update dns data fail, err:%v, domain:%s, sub:%s\n", err, *domain, *subDomain)
				break
			}
			fmt.Printf("Set new dns value to namesilo succ, recordid:%s, target ip:%s\n", data.RecordId, data.OutterIP)
			break
		}
		time.Sleep(time.Duration(*circle) * time.Second)
	}
}

func main() {
	flag.Parse()
	doCircle()
}
