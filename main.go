package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/42wim/hclencoder"
	"github.com/spf13/viper"
)

// toml input
type Ttask struct {
	Name       string
	Taskgroup  string
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
	Task      []Ttask
}

// Json convert structs
type Root struct {
	Job []JobInfo `hcl:"job"`
}

type JobInfo struct {
	Name        string       `hcl:",key"`
	Datacenters []string     `hcl:"datacenters"`
	Meta        Meta         `hcl:"meta"`
	Constraint  []Constraint `hcl:"constraint"`
	Update      Update       `hcl:"update"`
	Group       []GroupInfo  `hcl:"group"`
}

type Meta map[string]string

type Constraint struct {
	Attribute        string `hcl:"attribute" hcle:"omitempty"`
	Value            string `hcl:"value" hcle:"omitempty"`
	Operator         string `hcl:"operator" hcle:"omitempty"`
	DistinctHosts    bool   `hcl:"distinct_hosts" hcle:"omitempty"`
	DistinctProperty string `hcl:"distinct_property" hcle:"omitempty"`
}

type Update struct {
	Stagger     string `hcl:"stagger"`
	MaxParallel int    `hcl:"max_parallel"`
}

type GroupInfo struct {
	Name       string       `hcl:",key"`
	Count      int          `hcl:"count"`
	Constraint []Constraint `hcl:"constraint"`
	Restart    Restart      `hcl:"restart"`
	Task       []TaskInfo   `hcl:"task"`
}

type Restart struct {
	Interval string `hcl:"interval"`
	Attempts int    `hcl:"attempts"`
	Delay    string `hcl:"delay"`
	Mode     string `hcl:"mode"`
}

type TaskInfo struct {
	Name      string            `hcl:",key"`
	Meta      Meta              `hcl:"meta"`
	Driver    string            `hcl:"driver"`
	Config    Config            `hcl:"config"`
	Service   Service           `hcl:"service"`
	Env       map[string]string `hcl:"env"`
	Resources Resources         `hcl:"resources"`
}

type Config struct {
	AdvertiseIpv6Address bool              `hcl:"advertise_ipv6_address"`
	Image                string            `hcl:"image"`
	Args                 []string          `hcl:"args,omitempty"`
	Labels               map[string]string `hcl:"labels" hcle:"omitempty"`
	Volumes              []string          `hcl:"volumes" hcle:"omitempty"`
	Logging              map[string]string `hcl:"logging"`
}

type Service struct {
	Name        string   `hcl:"name"`
	Tags        []string `hcl:"tags" hcle:"omitempty"`
	Port        int      `hcl:"port"`
	AddressMode string   `hcl:"address_mode"`
	Check       Check    `hcl:"check"`
}

type Check struct {
	Name         string       `hcl:"name"`
	Port         int          `hcl:"port"`
	AddressMode  string       `hcl:"address_mode"`
	Type         string       `hcl:"type"`
	Path         string       `hcl:"path" hcle:"omitempty"`
	Interval     string       `hcl:"interval"`
	Timeout      string       `hcl:"timeout"`
	CheckRestart CheckRestart `hcl:"check_restart" hcle:"omitempty"`
}

type CheckRestart struct {
	Grace string `hcl:"grace" hcle:"omitempty"`
}

type Resources struct {
	Memory  int     `hcl:"memory"`
	CPU     int     `hcl:"cpu"`
	Network Network `hcl:"network"`
}

type Network struct {
	Mbits int `hcl:"mbits"`
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

func getTaskForGroup(tj *Tjob, taskgroupName string) []TaskInfo {
	ti := []TaskInfo{}
	for _, task := range tj.Task {
		if task.Taskgroup == taskgroupName {
			ti = append(ti, TaskInfo{
				Name:   tj.Job + "-" + task.Name,
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
			})
		}
	}
	return ti
}

func getGroupForJob(tj *Tjob) []GroupInfo {
	gi := []GroupInfo{}
	for _, tg := range tj.Taskgroup {
		gi = append(gi, GroupInfo{
			Name:  tj.Job + "-" + tg.Name,
			Count: tg.Count,
			Constraint: []Constraint{
				{DistinctHosts: true},
				{DistinctProperty: "${meta.datacenter}",
					Value: getDistinctDatacenter(&tg)},
			},
			Restart: Restart{
				Interval: "1m",
				Attempts: 5,
				Delay:    "10s",
				Mode:     "delay",
			},
			Task: getTaskForGroup(tj, tg.Name),
		})
	}
	return gi
}

func convertTomlToHcl(tj *Tjob) string {
	ji := []JobInfo{}
	ji = append(ji, JobInfo{
		Name:        tj.Job,
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
	})
	root := Root{ji}

	res, _ := hclencoder.Encode(root)
	return string(res)
}

func readconfig() {
	viper.SetConfigName("nomadgen")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
}
