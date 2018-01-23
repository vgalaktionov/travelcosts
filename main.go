package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"time"

	"gopkg.in/yaml.v2"
)

// Config is a struct for the config yaml parsing
type Config struct {
	AtWorkPattern  string                 `yaml:"at_work_pattern"`
	OutputFileName string                 `yaml:"output_file"`
	DefaultValues  map[string]interface{} `yaml:"default_values"`
	WorkingHours   struct {
		From int `yaml:"from"`
		To   int `yaml:"to"`
	} `yaml:"working_hours"`
	DateFormat  string `yaml:"date_format"`
	LogFileName string `yaml:"log_file"`
}

var c = Config{}

const day = 24 * time.Hour

func main() {
	readConfig()

	logs, err := os.OpenFile(c.LogFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening logfile: %v", err)
	}
	defer logs.Close()

	logsOutput := io.MultiWriter(os.Stdout, logs)
	log.SetOutput(logsOutput)

	logTravelCosts()
}

func readConfig() {
	home := os.Getenv("HOME")
	f, err := ioutil.ReadFile(fmt.Sprintf("%s/.travelcosts.config.yml", home))
	if err != nil {
		log.Fatal("Failed reading config yaml!")
	}
	err = yaml.Unmarshal([]byte(f), &c)
	if err != nil {
		log.Fatal("Failed parsing config yaml!")
	}
}

func logTravelCosts() {
	if loggedToday() || !withinWorkingHours() {
		return
	}

	_, err := os.Stat(c.OutputFileName)
	outputFileExists := os.IsNotExist(err)

	outputFile, err := os.OpenFile(c.OutputFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal("Failed opening output file!")
	}
	defer outputFile.Close()

	w := csv.NewWriter(outputFile)

	if !outputFileExists {
		outputHeaders := []string{"Datum"}
		for k := range c.DefaultValues {
			outputHeaders = append(outputHeaders, k)
		}
		w.Write(outputHeaders)
	}

	if atWork() {
		row := []string{time.Now().Format(c.DateFormat)}
		for k := range c.DefaultValues {
			row = append(row, c.DefaultValues[k].(string))
		}
		w.Write(row)
	}

	w.Flush()
}

func atWork() bool {
	cmd := exec.Command(
		"/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport",
		"-I",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	airportInfo, err := ioutil.ReadAll(stdout)
	if err != nil {
		log.Fatal(err)
	}

	ssidRegex := regexp.MustCompile(`\sSSID:\s+(?P<network>\S.*)\n`)
	match := ssidRegex.FindSubmatch(airportInfo)
	if match == nil {
		log.Fatal("Failed parsing SSID output!")
	}
	ssid := match[1]

	atWorkRegex := regexp.MustCompile(c.AtWorkPattern)
	atWork := atWorkRegex.Match(ssid)

	log.Printf("Network: %s, at work: %t", ssid, atWork)

	return atWork
}

func loggedToday() bool {
	f, err := os.Open(c.OutputFileName)
	if err != nil {
		log.Fatalf("Could not open %s, Error: %s", c.OutputFileName, err)
	}

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed reading %s", c.OutputFileName)
	}
	lastEntry := records[len(records)-1]
	if lastEntry[0] == "Datum" {
		return false
	}

	date, err := time.Parse(c.DateFormat, lastEntry[0])
	if err != nil {
		log.Fatalf("Failed parsing time %s with format %s", lastEntry[0], c.DateFormat)
	}
	if time.Now().Truncate(day).Equal(date.Truncate(day)) {
		return true
	}
	return false
}

func withinWorkingHours() bool {
	hour := time.Now().Hour()
	working := hour >= c.WorkingHours.From && hour <= c.WorkingHours.To
	log.Printf("Working: %t", working)
	return working
}
