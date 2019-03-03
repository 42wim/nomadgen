package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/42wim/hclencoder"
	"github.com/spf13/viper"
	"gopkg.in/alecthomas/kingpin.v2"
)

// toml input
type Ttask struct {
	Name          string
	Taskgroup     string
	NagiosSms     *int
	NagiosMail    *int
	Hostname      string
	Image         string
	Args          []string
	Command       string
	NoForcePull   bool
	Volumes       []string
	Inject        []string
	Port          int
	Tags          []string
	Porttype      string
	CheckPath     string
	Grace         string
	CPU           int
	Memory        int
	Firewall      string
	Labels        []string
	VaultPolicies []string
	VaultEnv      []string
	VaultInject   []string
	Env           []string
	Service       []Tservice
}

type Tgroup struct {
	Name       string
	Count      int
	Canary     int
	AutoRevert bool
}

type Tjob struct {
	Job          string
	Team         string
	Project      string
	Contact      string
	Tier         string
	Type         string
	Cron         string
	Taskgroup    []Tgroup
	Task         []Ttask
	Jenkins      Jenkins
	Mattermost   string
	NoBuildLabel bool
	Organization string
	AutoRevert   bool
}

type Tservice struct {
	Name      string
	Firewall  string
	Port      int
	PortType  string
	CheckPath string
	Tags      []string
	Grace     string
}

type Jenkins struct {
	AutoDeployTier  string
	DisableNomadgen bool
}

// Json convert structs
type Root struct {
	Job []JobInfo `hcl:"job"`
}

type JobInfo struct {
	Name        string       `hcl:",key"`
	Datacenters []string     `hcl:"datacenters"`
	Meta        Meta         `hcl:"meta"`
	Type        string       `hcl:"type" hcle:"omitempty"`
	Periodic    Periodic     `hcl:"periodic" hcle:"omitempty"`
	Constraint  []Constraint `hcl:"constraint"`
	Update      Update       `hcl:"update" hcle:"omitempty" `
	Group       []GroupInfo  `hcl:"group"`
}

type Periodic struct {
	Cron            string `hcl:"cron" hcle:"omitempty"`
	ProhibitOverlap bool   `hcl:"prohibit_overlap" hcle:"omitempty"`
}

type Meta map[string]string
type Env map[string]string

type Constraint struct {
	Attribute        string `hcl:"attribute" hcle:"omitempty"`
	Value            string `hcl:"value" hcle:"omitempty"`
	Operator         string `hcl:"operator" hcle:"omitempty"`
	DistinctHosts    bool   `hcl:"distinct_hosts" hcle:"omitempty"`
	DistinctProperty string `hcl:"distinct_property" hcle:"omitempty"`
}

type Update struct {
	Stagger     string `hcl:"stagger" hcle:"omitempty"`
	MaxParallel int    `hcl:"max_parallel" hcle:"omitempty"`
	Canary      int    `hcl:"canary" hcle:"omitempty"`
	AutoRevert  bool   `hcl:"auto_revert" hcle:"omitempty"`
}

type GroupInfo struct {
	Name       string       `hcl:",key"`
	Count      int          `hcl:"count"`
	Update     Update       `hcl:"update" hcle:"omitempty"`
	Constraint []Constraint `hcl:"constraint"`
	Restart    Restart      `hcl:"restart" hcle:"omitempty"`
	Task       []TaskInfo   `hcl:"task"`
}

type Restart struct {
	Interval string `hcl:"interval"`
	Attempts int    `hcl:"attempts"`
	Delay    string `hcl:"delay"`
	Mode     string `hcl:"mode"`
}

type TaskInfo struct {
	Name      string     `hcl:",key"`
	Meta      Meta       `hcl:"meta"`
	Template  []Template `hcl:"template"`
	Driver    string     `hcl:"driver"`
	Config    Config     `hcl:"config"`
	Service   []Service  `hcl:"service"`
	Vault     Vault      `hcl:"vault" hcle:"omitempty"`
	Env       Env        `hcl:"env" hcle:"omitempty"`
	Resources Resources  `hcl:"resources"`
}

type Template struct {
	ChangeMode     string `hcl:"change_mode" hcle:"omitempty"`
	ChangeSignal   string `hcl:"change_signal" hcle:"omitempty"`
	Data           string `hcl:"data,literal" hcle:"omitempty"`
	Destination    string `hcl:"destination" hcle:"omitempty"`
	Env            bool   `hcl:"env" hcle:"omitempty"`
	LeftDelimiter  string `hcl:"left_delimiter" hcle:"omitempty"`
	Perms          string `hcl:"perms" hcle:"omitempty"`
	RightDelimiter string `hcl:"right_delimiter" hcle:"omitempty"`
	Source         string `hcl:"source" hcle:"omitempty"`
	Splay          string `hcl:"splay" hcle:"omitempty"`
	VaultGrace     string `hcl:"vault_grace" hcle:"omitempty"`
}

type Config struct {
	AdvertiseIpv6Address bool              `hcl:"advertise_ipv6_address"`
	Image                string            `hcl:"image"`
	Command              string            `hcl:"command" hcle:"omitempty"`
	Hostname             string            `hcl:"hostname" hcle:"omitempty"`
	ForcePull            bool              `hcl:"force_pull" hcle:"omitempty"`
	Args                 []string          `hcl:"args,omitempty"`
	Labels               map[string]string `hcl:"labels" hcle:"omitempty"`
	Volumes              []string          `hcl:"volumes" hcle:"omitempty"`
	Logging              map[string]string `hcl:"logging"`
}

type Service struct {
	Name        string   `hcl:"name"`
	Tags        []string `hcl:"tags" hcle:"omitempty"`
	Port        int      `hcl:"port" hcle:"omitempty"`
	AddressMode string   `hcl:"address_mode"`
	Check       Check    `hcl:"check" hcle:"omitempty"`
}

type Check struct {
	Name          string       `hcl:"name"`
	Port          int          `hcl:"port"`
	AddressMode   string       `hcl:"address_mode"`
	Type          string       `hcl:"type"`
	Protocol      string       `hcl:"protocol" hcle:"omitempty"`
	Path          string       `hcl:"path" hcle:"omitempty"`
	Interval      string       `hcl:"interval"`
	Timeout       string       `hcl:"timeout"`
	TLSSkipVerify bool         `hcl:"tls_skip_verify" hcle:"omitempty"`
	CheckRestart  CheckRestart `hcl:"check_restart" hcle:"omitempty"`
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

type Vault struct {
	Policies []string `hcl:"policies" hcle:"omitempty"`
}

var silent = false
var fail_when_missing = true

const version = "1.0.3"

func main() {
	var (
		cInit        = kingpin.Command("init", "creates a template nomadgen.toml and jenkinsfile (if it doesn't exist)")
		initTeam     = cInit.Flag("team", "team name. (NOMADGEN_TEAM env)").Required().Short('t').Envar("NOMADGEN_TEAM").String()
		initProject  = cInit.Flag("project", "project name (NOMADGEN_PROJECT env)").Required().Short('p').Envar("NOMADGEN_PROJECT").String()
		initContact  = cInit.Flag("contact", "email addresses (NOMADGEN_CONTACT env)").Required().Short('c').Envar("NOMADGEN_CONTACT").String()
		initJenkins  = cInit.Flag("jenkins", "overwrite Jenkinsfile").Short('j').Bool()
		initBatch    = cInit.Flag("batch", "create a batch specific nomadgen.toml").Short('b').Bool()
		cWrite       = kingpin.Command("write", "creates/overwrites a project.nomad and Jenkinsfile (if not existing) based on nomadgen.toml")
		writeJenkins = cWrite.Flag("jenkins", "overwrite Jenkinsfile").Short('j').Bool()
	)
	kingpin.Command("info", "show info about nomadgen configuration")
	kingpin.Command("jenkins", "used by jenkins to create a project.nomad").Hidden()
	kingpin.HelpFlag.Short('h')
	kingpin.UsageTemplate(kingpin.LongHelpTemplate)
	switch kingpin.Parse() {
	case "init":
		if *initBatch {
			createBatchExample(*initTeam, *initProject, *initContact, true)
		} else {
			createExample(*initTeam, *initProject, *initContact, true)
		}
		readconfig()
		var tj Tjob
		viper.Unmarshal(&tj)
		createJenkins(parseJob(&tj), tj.Mattermost, tj.Jenkins, *initJenkins)
	case "write", "jenkins":
		// read toml
		if kingpin.Parse() == "jenkins" {
			silent = true
			fail_when_missing = false
		}
		readconfig()
		var tj Tjob
		// unmarshal into Tjob
		viper.Unmarshal(&tj)
		if kingpin.Parse() == "jenkins" && tj.Jenkins.DisableNomadgen {
			return
		}
		// convert toml to hcl
		output := convertTomlToHcl(&tj)
		ioutil.WriteFile("project.nomad", []byte(output), 0600)
		fmt.Println("project.nomad written.")
		createJenkins(parseJob(&tj), tj.Mattermost, tj.Jenkins, *writeJenkins)
	case "info":
		// read toml
		readconfig()
		var tj Tjob
		// unmarshal into Tjob
		viper.Unmarshal(&tj)
		fmt.Printf("nomadgen version: %s\n", version)
		if tj.Jenkins.DisableNomadgen == false {
			fmt.Println("Jenkins will generate project.nomad on each build")
		} else {
			fmt.Println("Jenkins will not generate project.nomad on each build.\nDo not forget to run nomadgen write manually after every nomadgen.toml change")
		}
		if tj.Jenkins.AutoDeployTier != "" {
			fmt.Printf("Jenkins will automatically submit project.nomad to tier %s\n", tj.Jenkins.AutoDeployTier)
		}
	}
}

func parseOrganization(tj *Tjob) string {
	if tj.Organization == "" {
		return "prefix"
	}
	return tj.Organization
}

func parseJob(tj *Tjob) string {
	job := ""
	job = parseOrganization(tj)
	job += "-${short_tier}-"
	job += tj.Team + "-" + tj.Project
	return job
}

func parseEnv(tj *Tjob, labels []string) Env {
	lmap := make(Env)
	for _, label := range labels {
		strs := strings.SplitN(label, "=", 2)
		lmap[strings.TrimSpace(strs[0])] = strings.TrimSpace(strs[1])
	}
	// return uninitialized if we have no keys
	if len(lmap) == 0 {
		return nil
	}
	return lmap
}

func parseLabels(tj *Tjob, labels []string) map[string]string {
	lmap := parseEnv(tj, labels)
	// if we have an empty result, add it
	if lmap == nil {
		lmap = make(Env)
	}
	if !tj.NoBuildLabel {
		lmap["lnx_build"] = "${BUILD_NUMBER}"
	}
	return lmap
}

func parseInject(tj *Tjob, inject []string) ([]Template, []string) {
	var templates []Template
	var volumes []string
	// make map unique
	m := make(map[string]bool)
	for _, v := range inject {
		m[v] = true
	}
	// auto-inject vault.env if it exists
	if _, err := os.Stat("vault.env"); err == nil {
		ok := true
		for k := range m {
			if strings.HasPrefix(k, "vault.env:") {
				ok = false
			}
		}
		if ok {
			m["vault.env"] = true
		}
	}

	for input := range m {
		splitInput := strings.Split(input, ":")
		fileName := splitInput[0]
		envOpt := false
		if len(splitInput) > 1 {
			for _, opt := range splitInput[1:] {
				if opt == "env" {
					envOpt = true
				}
				// if we have a / starting it means we want to overwrite a file in the container
				// append this to the volumes []string with the full path
				if strings.HasPrefix(opt, "/") {
					volumes = append(volumes, "secrets/"+fileName+":"+opt)
				}
			}
		}
		if strings.HasSuffix(fileName, ".env") {
			envOpt = true
		}
		content, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Println("Something went wrong:")
			log.Println(err)
			continue
		}
		templates = append(templates, Template{Data: "<<EOH\n" + string(content) + "\nEOH", Destination: "secrets/" + fileName, Env: envOpt})
	}
	return templates, volumes
}

// createVaultEnvInject creates vault-taskname-count.env files which contain the necessary information
// to be injected in the template stanza. It returns the created filename.
func createVaultEnvInject(tj *Tjob, env []string, name string, count int) string {
	content := ""
	for _, e := range env {
		fp := tj.Project + "/" + e
		fb := filepath.Base(e)
		if strings.Contains(e, "<-") {
			res := strings.Split(e, "<-")
			e = res[1]
			fb = res[0]
			fp = tj.Project + "/" + e
		}
		if strings.Contains(e, "/") {
			fp = e
		}
		content += fb + "=\"{{with secret \"secret/projects/prefix-${short_tier}-" + tj.Team + "/" + fp + "\"}}{{.Data.value}}{{end}}\"\n"
	}
	// return nothing if we have no content
	if content == "" {
		return ""
	}
	f := "vault-" + name + "-" + strconv.Itoa(count) + ".env"
	ioutil.WriteFile(f, []byte(content), 0600)
	return f
}

// createVaultFileInject creates vault-taskname-count.inj files which contain the necessary information
// to be injected in the template stanza. It returns the created filename.
func createVaultFileInject(tj *Tjob, files []string, name string, count int) []string {
	result := []string{}
	content := ""
	i := 0
	for _, entry := range files {
		splitInput := strings.Split(entry, ":")
		vaultKey := splitInput[0]
		content = "{{with secret \"secret/projects/prefix-${short_tier}-" + tj.Team + "/" + tj.Project + "/" + vaultKey + "\"}}{{.Data.value}}{{end}}\n"
		f := "vault-" + name + "-" + strconv.Itoa(count) + "-" + strconv.Itoa(i) + ".inj"
		ioutil.WriteFile(f, []byte(content), 0600)
		if len(splitInput) > 0 {
			result = append(result, f+":"+splitInput[1])
		} else {
			result = append(result, f)
		}
		i++
	}
	// return nothing if we have no content
	if content == "" {
		return result
	}
	return result
}

func getPeriodic(tj *Tjob) Periodic {
	if tj.Type != "batch" {
		return Periodic{}
	}
	// return empty if we have a batch without a cron
	if tj.Cron == "" {
		return Periodic{}
	}
	cron := strings.Split(tj.Cron, ":")
	if strings.Contains(tj.Cron, ":allow_overlap") {
		return Periodic{Cron: cron[0], ProhibitOverlap: false}
	}
	return Periodic{Cron: cron[0], ProhibitOverlap: true}
}

func getRestart(tj *Tjob) Restart {
	if tj.Type == "batch" {
		return Restart{}
	}
	return Restart{Interval: "1m", Attempts: 5, Delay: "10s", Mode: "delay"}
}

func getUpdate(tj *Tjob) Update {
	if tj.Type == "batch" {
		return Update{}
	}
	return Update{Stagger: "10s", MaxParallel: 1, AutoRevert: tj.AutoRevert}
}

func getVault(tj *Tjob, policies []string) Vault {
	v := Vault{}
	policies = getVaultPolicies(tj, policies)
	if len(policies) == 0 {
		return v
	}
	v.Policies = policies
	return v
}

func getVaultPolicies(tj *Tjob, policies []string) []string {
	n := []string{}
	prefix := parseOrganization(tj)
	for _, policy := range policies {
		n = append(n, prefix+"-${short_tier}-"+tj.Team+"-"+policy)
	}
	return n
}

func getTaskName(tj *Tjob, input interface{}) string {
	switch input.(type) {
	case Tservice:
		return getServiceName(tj, input.(Tservice).Name)
	}
	task := input.(Ttask)
	if task.Name == "" {
		return parseJob(tj)
	}
	return parseJob(tj) + "-" + task.Name
}

func getServiceName(tj *Tjob, name string) string {
	if name == "" {
		return parseJob(tj)
	}
	return parseJob(tj) + "-" + name
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

func getFirewallForService(tj *Tjob, task Ttask) Env {
	var env []Env
	if len(task.Service) > 0 {
		for _, svc := range task.Service {
			task := Ttask{Porttype: svc.PortType, Firewall: svc.Firewall, CheckPath: svc.CheckPath, Grace: svc.Grace, Port: svc.Port, Tags: svc.Tags, Name: svc.Name, Env: task.Env}
			env = append(env, getFirewall(tj, task))
		}
	} else {
		env = append(env, getFirewall(tj, task))
	}
	resultEnv := make(Env)
	for _, e := range env {
		for k, v := range e {
			resultEnv[k] = v
		}
	}
	// return uninitialized if we have no keys
	if len(resultEnv) == 0 {
		return nil
	}
	return resultEnv
}

func getFirewall(tj *Tjob, task Ttask) Env {
	env := parseEnv(tj, task.Env)
	if task.Port == 0 {
		if len(env) == 0 {
			return nil
		}
		return env
	}
	res := []string{}
	fws := strings.Split(task.Firewall, ",")
	prefix := parseOrganization(tj)
	for _, fw := range fws {
		if strings.HasPrefix(fw, "xs/") {
			fw = strings.Replace(fw, "xs/", "", -1)
			res = append(res, "s/"+prefix+"-${short_tier}-"+fw)
		} else if strings.HasPrefix(fw, "xg/") {
			fw = strings.Replace(fw, "xg/", "", -1)
			res = append(res, "g/"+prefix+"-${short_tier}-"+fw)
		} else {
			res = append(res, fw)
		}
	}
	fw := strings.Join(res, ",")
	// initialize env if we have nothing yet
	if fw != "" && env == nil {
		env = make(Env)
	}
	env["FIREWALL_"+strconv.Itoa(task.Port)] = fw
	return env
}

func isemptyFirewall(tj *Tjob, task Ttask) bool {
	if task.Firewall == "" {
		return true
	}
	fws := strings.Split(task.Firewall, ",")
	for _, fw := range fws {
		if fw == "s/none" {
			return true
		}
	}
	return false
}

func getCheck(tj *Tjob, task Ttask) Check {
	if isemptyFirewall(tj, task) {
		return Check{}
	}
	if task.Porttype == "none" {
		return Check{}
	}
	if task.Porttype == "" {
		task.Porttype = "tcp"
	}
	checktype := task.Porttype
	protocol := ""
	tlsSkipVerify := false
	if strings.HasPrefix(task.Porttype, "https") {
		checktype = "http"
		protocol = "https"
		if strings.Contains(task.Porttype, ":tls_skip_verify") {
			tlsSkipVerify = true
		}
	}
	check := Check{Name: getTaskName(tj, task) + "-check",
		Port:          task.Port,
		AddressMode:   "driver",
		Type:          checktype,
		Protocol:      protocol,
		TLSSkipVerify: tlsSkipVerify,
		Path:          task.CheckPath,
		Interval:      "20s",
		Timeout:       "10s",
		CheckRestart: CheckRestart{
			Grace: task.Grace,
		},
	}
	return check
}

func getTier(tj *Tjob) string {
	if tj.Tier == "" {
		return "${long_tier}"
	}
	return ""
}

func getDistinctDatacenter(tg *Tgroup) string {
	if tg.Count == 0 || tg.Count == 1 {
		return "1"
	}
	if tg.Count%2 != 0 {
		log.Fatalf("group count %s is odd: %d", tg.Name, tg.Count)
	}
	// if canary is not even, we make it even for the distinct property count
	if tg.Canary%2 != 0 {
		return strconv.Itoa((tg.Canary + 1 + tg.Count) / 2)
	}
	distinct := (tg.Count + tg.Canary) / 2
	return strconv.Itoa(distinct)
}

func getServiceForTask(tj *Tjob, task Ttask) []Service {
	var services []Service
	if len(task.Service) > 0 {
		for _, svc := range task.Service {
			task := Ttask{Porttype: svc.PortType, Firewall: svc.Firewall, CheckPath: svc.CheckPath, Grace: svc.Grace, Port: svc.Port, Tags: svc.Tags, Name: svc.Name}
			services = append(services, Service{
				Name:        getTaskName(tj, task),
				Port:        svc.Port,
				Tags:        svc.Tags,
				AddressMode: "driver",
				Check:       getCheck(tj, task),
			})
		}
	} else {
		services = []Service{{
			Name:        getTaskName(tj, task),
			Port:        task.Port,
			Tags:        task.Tags,
			AddressMode: "driver",
			Check:       getCheck(tj, task),
		}}
	}
	return services
}

func getTaskForGroup(tj *Tjob, taskgroupName string) []TaskInfo {
	ti := []TaskInfo{}
	for i, task := range tj.Task {
		if task.Taskgroup == taskgroupName {
			result := createVaultEnvInject(tj, task.VaultEnv, taskgroupName, i)
			if result != "" {
				task.Inject = append(task.Inject, result)
			}
			results := createVaultFileInject(tj, task.VaultInject, taskgroupName, i)
			if len(result) > 0 {
				task.Inject = append(task.Inject, results...)
			}
			if isemptyFirewall(tj, task) {
				task.Port = 0
			}
			templates, volumes := parseInject(tj, task.Inject)
			if len(volumes) > 0 {
				task.Volumes = append(task.Volumes, volumes...)
			}
			ti = append(ti, TaskInfo{
				Name:     getTaskName(tj, task),
				Meta:     getTaskMeta(task),
				Driver:   "docker",
				Template: templates,
				Config: Config{
					AdvertiseIpv6Address: true,
					Image:                task.Image,
					Args:                 task.Args,
					Hostname:             task.Hostname,
					Command:              task.Command,
					ForcePull:            !task.NoForcePull,
					Volumes:              task.Volumes,
					Labels:               parseLabels(tj, task.Labels),
					Logging:              map[string]string{"type": "journald"},
				},
				Service: getServiceForTask(tj, task),
				Env:     getFirewallForService(tj, task),
				Vault:   getVault(tj, task.VaultPolicies),
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
			Name:  parseJob(tj) + "-" + tg.Name,
			Count: tg.Count,
			Constraint: []Constraint{
				{DistinctHosts: true},
				{DistinctProperty: "${meta.datacenter}",
					Value: getDistinctDatacenter(&tg)},
			},
			Restart: getRestart(tj),
			Update: Update{
				Canary:     tg.Canary,
				AutoRevert: tg.AutoRevert,
			},
			Task: getTaskForGroup(tj, tg.Name),
		})
	}
	return gi
}

func convertTomlToHcl(tj *Tjob) string {
	ji := []JobInfo{}
	ji = append(ji, JobInfo{
		Name:        parseJob(tj),
		Type:        tj.Type,
		Periodic:    getPeriodic(tj),
		Datacenters: []string{datacenter},
		Meta:        Meta{"contact": tj.Contact},
		Constraint: []Constraint{
			{Attribute: "${meta.role}",
				Value: metarole},
			{Attribute: "${meta.tier}",
				Value: getTier(tj)},
		},
		Update: getUpdate(tj),
		Group:  getGroupForJob(tj),
	})
	root := Root{ji}

	res, _ := hclencoder.Encode(root)
	return string(res)
}

func readconfig() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("nomadgen")
	viper.SetConfigName("nomadgen")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		fail := true
		if strings.Contains(err.Error(), "Not Found") {
			if !silent {
				fmt.Printf("error: config file: %s\nrun nomadgen --init\n", err)
			}
			fail = fail_when_missing
		} else {
			fmt.Printf("error: config file: %s\n", err)
		}
		if fail {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}
	if !strings.HasSuffix(viper.ConfigFileUsed(), ".toml") {
		fmt.Fprintln(os.Stderr, "Only toml is officially suported. Contact jo vandeginste for problems with other input formats.")
	}
}
