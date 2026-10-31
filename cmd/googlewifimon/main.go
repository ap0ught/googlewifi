package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bengalaviz/googlewifi/monitor"
)

type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

type statusResponse struct {
	System struct {
		Role string `json:"groupRole"`
		LED  string `json:"ledAnimation"`
	} `json:"system"`
}

type database struct {
	Hosts   map[string]monitor.HostState `json:"hosts"`
	Events  []event                      `json:"events"`
	Samples map[string][]sample          `json:"samples"`
}
type event struct {
	Host string `json:"host"`
	monitor.Event
}
type sample struct {
	At        time.Time     `json:"at"`
	State     monitor.State `json:"state"`
	PingLoss  float64       `json:"ping_loss_percent"`
	PingAvgMS float64       `json:"ping_avg_ms"`
}

var machineOutput bool

var pingRE = regexp.MustCompile(`([0-9.]+)% packet loss`)
var rttRE = regexp.MustCompile(`= [0-9.]+/([0-9.]+)/`)

func main() {
	var hosts stringList
	flag.Var(&hosts, "host", "device IP or URL; repeat for multiple devices")
	statePath := flag.String("state", defaultStatePath(), "JSON state/history file")
	interval := flag.Duration("interval", 0, "repeat checks at this interval; 0 runs once")
	pingCount := flag.Int("ping-count", 5, "ICMP packets per check")
	jsonOutput := flag.Bool("json", false, "print the complete state as JSON")
	flag.Parse()
	machineOutput = *jsonOutput
	if len(hosts) == 0 {
		hosts = stringList{"192.168.86.1", "192.168.86.23"}
	}
	if *pingCount < 1 {
		fail("-ping-count must be positive")
	}

	for {
		db := load(*statePath)
		if db.Hosts == nil {
			db.Hosts = map[string]monitor.HostState{}
		}
		if db.Samples == nil {
			db.Samples = map[string][]sample{}
		}
		now := time.Now().UTC()
		worst := monitor.StateOK
		for _, host := range hosts {
			if result := check(host, &db, now, *pingCount); severity(result.State) > severity(worst) {
				worst = result.State
			}
		}
		if err := save(*statePath, db); err != nil {
			fail("saving state: %v", err)
		}
		if machineOutput {
			if err := json.NewEncoder(os.Stdout).Encode(db); err != nil {
				fail("encoding state: %v", err)
			}
		}
		if *interval <= 0 {
			os.Exit(exitCode(worst))
		}
		time.Sleep(*interval)
	}
}

func check(host string, db *database, now time.Time, pingCount int) monitor.Result {
	ip := host
	if strings.Contains(host, "://") {
		ip = strings.SplitN(strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://"), "/", 2)[0]
	}
	input := monitor.StatusInput{HTTPStatus: 0}
	var status statusResponse
	url := host
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}
	if !strings.HasSuffix(url, "/api/v1/status") {
		url = strings.TrimRight(url, "/") + "/api/v1/status"
	}
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err == nil {
		input.HTTPStatus = resp.StatusCode
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			err = json.NewDecoder(resp.Body).Decode(&status)
		}
		resp.Body.Close()
	}
	input.Err, input.Role, input.LED = err, status.System.Role, status.System.LED
	input.PingLoss, input.PingAvgMS = ping(ip, pingCount)
	result := monitor.Evaluate(input)
	old := db.Hosts[host]
	change := monitor.Condense(&old, result, now)
	db.Hosts[host] = old
	db.Samples[host] = append(db.Samples[host], sample{At: now, State: result.State, PingLoss: input.PingLoss, PingAvgMS: input.PingAvgMS})
	if len(db.Samples[host]) > 1440 {
		db.Samples[host] = db.Samples[host][len(db.Samples[host])-1440:]
	}
	if change != nil {
		db.Events = append(db.Events, event{Host: host, Event: *change})
		if len(db.Events) > 1000 {
			db.Events = db.Events[len(db.Events)-1000:]
		}
		if !machineOutput {
			fmt.Printf("CHANGE %s %s -> %s - %s\n", host, change.Previous, change.State, change.Reason)
		}
	} else if !machineOutput {
		fmt.Printf("%s %s - %s (checks=%d consecutive=%d)\n", host, result.State, result.Reason, old.Checks, old.Consecutive)
	}
	return result
}

func severity(state monitor.State) int {
	switch state {
	case monitor.StateCritical:
		return 3
	case monitor.StateWarning:
		return 2
	case monitor.StateUnknown:
		return 1
	default:
		return 0
	}
}

func exitCode(state monitor.State) int {
	if state == monitor.StateCritical {
		return 2
	}
	if state == monitor.StateWarning || state == monitor.StateUnknown {
		return 1
	}
	return 0
}

func ping(host string, count int) (float64, float64) {
	out, err := exec.Command("ping", "-c", strconv.Itoa(count), "-W", "1", host).CombinedOutput()
	if err != nil && len(out) == 0 {
		return 100, 0
	}
	text := string(out)
	loss, avg := 100.0, 0.0
	if m := pingRE.FindStringSubmatch(text); len(m) == 2 {
		loss, _ = strconv.ParseFloat(m[1], 64)
	}
	if m := rttRE.FindStringSubmatch(text); len(m) == 2 {
		avg, _ = strconv.ParseFloat(m[1], 64)
	}
	return loss, avg
}

func load(path string) database {
	data, err := os.ReadFile(path)
	if err != nil {
		return database{}
	}
	var db database
	if json.Unmarshal(data, &db) != nil {
		return database{}
	}
	return db
}
func save(path string, db database) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
func defaultStatePath() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "googlewifimon-state.json"
	}
	return filepath.Join(h, ".local", "state", "googlewifimon", "state.json")
}
func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
