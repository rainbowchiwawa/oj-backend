package parser

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v2"
)

type ProblemTestcase struct {
	Score int `yaml:"score" json:"score"`
}

type ProblemPublicPair struct {
	Source string `yaml:"source" json:"source"`
	Target string `yaml:"target" json:"target"`
}

type ProblemSettings struct {
	Title  string `yaml:"title" json:"-"`
	Limits struct {
		TotalTime int `yaml:"totalTime" json:"total_time"`
		CPUTime   int `yaml:"cpuTime" json:"cpu_time"`
		Memory    int `yaml:"memory" json:"memory"`
	} `yaml:"limits" json:"limits"`
	Tests  []ProblemTestcase   `yaml:"presets" json:"tests"`
	Public []ProblemPublicPair `yaml:"public" json:"public"`
}

func (ps ProblemSettings) Value() (driver.Value, error) {
	return json.Marshal(ps)
}

func (ps *ProblemSettings) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Failed to unmarshal JSONB")
	}
	return json.Unmarshal(bytes, ps)
}

func ParseProblemSettings(bytes []byte) (*ProblemSettings, error) {
	var settings ProblemSettings
	if err := yaml.Unmarshal(bytes, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}
