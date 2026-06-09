package parser

import (
	"encoding/xml"
	"time"
)

type TestResults struct {
	XMLName   xml.Name  `xml:"testsuite" json:"-"`
	Name      string    `xml:"name" json:"name"`
	Tests     int       `xml:"tests" json:"-"`
	Failures  int       `xml:"failures" json:"-"`
	Disabled  int       `xml:"disabled" json:"-"`
	Skipped   int       `xml:"skipped" json:"-"`
	Hostname  string    `xml:"hostname" json:"-"`
	Time      int       `xml:"time" json:"time"`
	Timestamp time.Time `xml:"timestamp" json:"-"`
	Testcases []struct {
		Name      string  `xml:"name" json:"name"`
		ClassName string  `xml:"classname" json:"-"`
		Time      float64 `xml:"time" json:"time"`
		Status    string  `xml:"status" json:"status"`
		Failure   *struct {
			Message string `xml:"message" json:"message"`
		} `xml:"failure" json:"failure"`
		SystemOut struct {
			Content string `xml:",chardata" json:"content"`
		} `xml:"system-out" json:"system_out"`
	} `xml:"testcase" json:"testcases"`
}

func ParseTestResults(bytes []byte) (*TestResults, error) {
	var results TestResults
	if err := xml.Unmarshal(bytes, &results); err != nil {
		return nil, err
	}
	return &results, nil
}
