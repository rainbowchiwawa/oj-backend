package parser

import (
	"database/sql/driver"
	"encoding/json"
	"encoding/xml"
	"fmt"
)

type TestResults struct {
	XMLName   xml.Name  `xml:"testsuite" json:"-"`
	Name      string    `xml:"name,attr" json:"name"`
	Tests     int       `xml:"tests,attr" json:"-"`
	Failures  int       `xml:"failures,attr" json:"-"`
	Disabled  int       `xml:"disabled,attr" json:"-"`
	Skipped   int       `xml:"skipped,attr" json:"-"`
	Hostname  string    `xml:"hostname,attr" json:"-"`
	Time      int       `xml:"time,attr" json:"time"`
	Timestamp string    `xml:"timestamp,attr" json:"-"`
	Testcases []struct {
		Name      string  `xml:"name,attr" json:"name"`
		ClassName string  `xml:"classname,attr" json:"-"`
		Time      float64 `xml:"time,attr" json:"time"`
		Status    string  `xml:"status,attr" json:"status"`
		Failure   *struct {
			Message string `xml:"message,attr" json:"message"`
		} `xml:"failure" json:"failure"`
		SystemOut struct {
			Content string `xml:",chardata" json:"content"`
		} `xml:"system-out" json:"system_out"`
	} `xml:"testcase" json:"testcases"`
}

func (tr TestResults) Value() (driver.Value, error) {
	return json.Marshal(tr)
}

func (tr *TestResults) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Failed to unmarshal JSONB")
	}
	return json.Unmarshal(bytes, tr)
}

func ParseTestResults(bytes []byte) (*TestResults, error) {
	var results TestResults
	if err := xml.Unmarshal(bytes, &results); err != nil {
		return nil, err
	}
	return &results, nil
}
