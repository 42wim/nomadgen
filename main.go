package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/hcl/printer"
	jsonParser "github.com/hashicorp/hcl/json/parser"
	"github.com/spf13/viper"
)

// toml input
type Ttask struct {
	Name       string
	NagiosSms  *int
	NagiosMail *int
	Image      string
	Args       []string
	Volumes    []string
	Port       int
	Tags       []string
	Porttype   string
	CheckPath  string
	Grace      string
	CPU        int
	Memory     int
	Firewall   string
}

type Tgroup struct {
	Name  string
	Count int
}

type Tjob struct {
	Job       string
	Contact   string
	Tier      string
	Taskgroup []Tgroup
	Task      map[string][]Ttask
}

// Json convert structs
type Root struct {
	Job map[string]JobInfo `json:"job"`
}

type JobInfo struct {
	Datacenters []string             `json:"datacenters"`
	Meta        Meta                 `json:"meta"`
	Constraint  []Constraint         `json:"constraint"`
	Update      Update               `json:"update"`
	Group       map[string]GroupInfo `json:"group"`
}

type Meta map[string]string

type Constraint struct {
	Attribute string `json:"attribute,omitempty"`
	Value     string `json:"value"`
	Operator  string `json:"operator,omitempty"`
}

type Update struct {
	Stagger     string `json:"stagger"`
	MaxParallel int    `json:"max_parallel"`
}

type GroupInfo struct {
	Count      int                 `json:"count"`
	Constraint []Constraint        `json:"constraint"`
	Restart    Restart             `json:"restart"`
	Task       map[string]TaskInfo `json:"task"`
}

type Restart struct {
	Interval string `json:"interval"`
	Attempts int    `json:"attempts"`
	Delay    string `json:"delay"`
	Mode     string `json:"mode"`
}

type TaskInfo struct {
	Meta      Meta              `json:"meta"`
	Driver    string            `json:"driver"`
	Config    Config            `json:"config"`
	Service   Service           `json:"service"`
	Env       map[string]string `json:"env"`
	Resources Resources         `json:"resources"`
}

type Config struct {
	Image                string            `json:"image"`
	Args                 []string          `json:"args,omitempty"`
	AdvertiseIpv6Address bool              `json:"advertise_ipv6_address"`
	Labels               map[string]string `json:"labels,omitempty"`
	Volumes              []string          `json:"volumes,omitempty"`
	Logging              map[string]string `json:"logging"`
}

type Service struct {
	Name        string   `json:"name"`
	Port        int      `json:"port"`
	AddressMode string   `json:"address_mode"`
	Check       Check    `json:"check"`
	Tags        []string `json:"tags,omitempty"`
}

type Check struct {
	Name         string       `json:"name"`
	Port         int          `json:"port"`
	AddressMode  string       `json:"address_mode"`
	Type         string       `json:"type"`
	Path         string       `json:"path,omitempty"`
	Interval     string       `json:"interval"`
	Timeout      string       `json:"timeout"`
	CheckRestart CheckRestart `json:"check_restart,omitempty"`
}

type CheckRestart struct {
	Grace string `json:"grace,omitempty"`
}

type Resources struct {
	Memory  int     `json:"memory"`
	CPU     int     `json:"cpu"`
	Network Network `json:"network"`
}

type Network struct {
	Mbits int `json:"mbits"`
}

func main() {
	// read toml
	readconfig()
	var tj Tjob
	// unmarshal into Tjob
	viper.Unmarshal(&tj)
	// convert toml to hcl
	output := convertTomlToHcl(&tj)
	fmt.Println(output)
}

func getTaskMeta(task Ttask) Meta {
	meta := make(Meta)
	if task.NagiosMail == nil {
		meta["nagios_mail"] = "-1"
	} else {
		meta["nagios_mail"] = strconv.Itoa(*task.NagiosMail)
	}
	if task.NagiosSms == nil {
		meta["nagios_sms"] = "-1"
	} else {
		meta["nagios_sms"] = strconv.Itoa(*task.NagiosSms)
	}
	return meta
}

func getFirewall(task Ttask) map[string]string {
	env := make(map[string]string)
	if task.Port == 0 {
		return env
	}
	env["FIREWALL_"+strconv.Itoa(task.Port)] = task.Firewall
	return env
}

func getTier(tj *Tjob) string {
	if tj.Tier != "" {
		return tj.Tier
	}
	s := strings.Split(tj.Job, "-")
	if len(s) > 2 {
		switch s[1] {
		case "p":
			return "production"
		case "q", "t":
			return "quality"
		}
	}
	log.Fatal("no tier found")
	return ""
}

func getDistinctDatacenter(tg *Tgroup) string {
	if tg.Count == 0 || tg.Count == 1 {
		return "1"
	}
	if tg.Count%2 != 0 {
		log.Fatalf("group count %s is odd", tg.Name)
	}
	distinct := tg.Count / 2
	return strconv.Itoa(distinct)
}

func getTaskForGroup(tj *Tjob, taskgroupName string) map[string]TaskInfo {
	ti := make(map[string]TaskInfo)
	for _, task := range tj.Task[taskgroupName] {
		ti[tj.Job+"-"+task.Name] = TaskInfo{
			Meta:   getTaskMeta(task),
			Driver: "docker",
			Config: Config{
				AdvertiseIpv6Address: true,
				Image:                task.Image,
				Args:                 task.Args,
				Volumes:              task.Volumes,
				Logging:              map[string]string{"type": "journald"},
			},
			Service: Service{
				Name:        tj.Job + "-" + task.Name,
				Port:        task.Port,
				Tags:        task.Tags,
				AddressMode: "driver",
				Check: Check{
					Name:        tj.Job + "-" + task.Name + "-check",
					Port:        task.Port,
					AddressMode: "driver",
					Type:        task.Porttype,
					Path:        task.CheckPath,
					Interval:    "20s",
					Timeout:     "10s",
					CheckRestart: CheckRestart{
						Grace: task.Grace,
					},
				},
			},
			Env: getFirewall(task),
			Resources: Resources{
				Memory: task.Memory,
				CPU:    task.CPU,
				Network: Network{
					Mbits: 1,
				},
			},
		}
	}
	return ti
}

func getGroupForJob(tj *Tjob) map[string]GroupInfo {
	gi := make(map[string]GroupInfo)
	for _, tg := range tj.Taskgroup {
		gi[tg.Name] = GroupInfo{
			Count: tg.Count,
			Constraint: []Constraint{
				{Operator: "distinct_hosts",
					Value: "true"},
				{Operator: "distinct_property",
					Attribute: "${meta.datacenter}",
					Value:     getDistinctDatacenter(&tg)},
			},
			Restart: Restart{
				Interval: "1m",
				Attempts: 5,
				Delay:    "10s",
				Mode:     "delay",
			},
			Task: getTaskForGroup(tj, tg.Name),
		}
	}
	return gi
}

func convertTomlToHcl(tj *Tjob) string {
	ji := make(map[string]JobInfo)
	ji[tj.Job] = JobInfo{
		Datacenters: []string{datacenter},
		Meta:        Meta{"contact": tj.Contact},
		Constraint: []Constraint{
			{Attribute: "${meta.role}",
				Value: metarole},
			{Attribute: "${meta.tier}",
				Value: getTier(tj)},
		},
		Update: Update{
			Stagger:     "10s",
			MaxParallel: 1,
		},
		Group: getGroupForJob(tj),
	}
	root := Root{ji}

	res, _ := json.Marshal(&root)
	ast, _ := jsonParser.Parse(res)
	buf := new(bytes.Buffer)
	printer.Fprint(buf, ast)

	// beautify the output
	lines := strings.Split(buf.String(), "\n")
	newlines := ""
	for _, line := range lines {
		if line != "" {
			newlines += line + "\n"
		}
	}
	res, _ = printer.Format([]byte(newlines))
	return string(res)
}

func readconfig() {
	viper.SetConfigName("nomadgen")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
}
