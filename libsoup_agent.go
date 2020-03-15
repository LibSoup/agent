package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var flagHostname = flag.String("n", "", "Override the discovered hostname")
var flagUploadURL = flag.String("a", "https://libsoup.com/api/v1/upload", "LibSoup server address")
var flagTransferID = flag.String("id", "", "Agent transfer ID")

type hostData struct {
	TxID     string
	Hostname string
	Libs     map[string]int
	Os       string
}

func main() {
	flag.Parse()
	var data hostData

	data.TxID = *flagTransferID
	data.Os = discoverOs()
	data.Libs = uniqueLibs(analyzeProcs())

	if *flagHostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			log.Print("ERROR geting hostname: " + err.Error())
			return
		}
		data.Hostname = hostname
	} else {
		data.Hostname = *flagHostname
	}

	uploadData(data)
}

// Build a map of unique library filenames and the # of instances open
func uniqueLibs(libs []string) map[string]int {
	ul := make(map[string]int)
	for _, lib := range libs {
		if _, ok := ul[lib]; ok {
			i := ul[lib]
			ul[lib] = i + 1
		} else {
			ul[lib] = 1
		}
	}

	return ul
}

// This will need to expand some
func discoverOs() string {
	return osFromRelease()
}

// Read OS name from the /etc/os-release file (works for most of the systems tested)
func osFromRelease() string {
	out, err := exec.Command("grep", "-E", "^NAME", "/etc/os-release").Output()
	if err != nil {
		log.Print("ERROR getting OS name: " + err.Error())
		return ""
	}

	str := strings.TrimPrefix(string(out), "NAME=")
	str = strings.TrimSpace(str)
	str = strings.Trim(str, "\"")
	return str
}

// Process the contents of /proc
func analyzeProcs() []string {
	var libs []string

	procfiles, err := ioutil.ReadDir("/proc")
	if err != nil {
		log.Print("ERROR reading /proc: " + err.Error())
		return libs
	}

	// analyze each process (don't count the same exe > once)
	var analyzed = make(map[string]bool)
	for _, dir := range procfiles {
		if dir.IsDir() {
			pexe, plibs := getProcLibs(dir.Name())
			if _, ok := analyzed[pexe]; ok {
				continue
			}

			libs = append(libs, plibs...)
			analyzed[pexe] = true
		}
	}
	return libs
}

// Analyze linked librar
func getProcLibs(pid string) (string, []string) {
	var libs []string
	re := regexp.MustCompile(`.+ => (.+) .+`)

	// ignore ReadLink error since exec.Command will fail anyway
	linkdest, _ := os.Readlink("/proc/" + pid + "/exe")
	out, err := exec.Command("ldd", linkdest).Output()
	if err != nil {
		return linkdest, libs
	}

	matches := re.FindAllStringSubmatch(string(out), -1)
	for _, match := range matches {
		if match[1] == "not" {
			continue
		}
		libs = append(libs, filepath.Base(match[1]))
	}

	return linkdest, libs
}

// Send the hostData object to the libsoup.com servers
func uploadData(data hostData) {
	out, oerr := json.Marshal(data)
	if oerr != nil {
		log.Print("ERROR serializing host data: " + oerr.Error())
		return
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(out)
	if err != nil {
		log.Print("ERROR compressing data: " + err.Error())
		return
	}

	if err := zw.Close(); err != nil {
		log.Print("ERROR closing gzip writer: " + err.Error())
		return
	}

	log.Printf("Sending  hostname=%s, os=%s, libcount=%d", data.Hostname, data.Os, len(data.Libs))

	req, err := http.NewRequest("POST", *flagUploadURL, &buf)
	if err != nil {
		log.Print("ERROR creating http request: " + err.Error())
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Encoding", "gzip")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, rerr := client.Do(req)
	if rerr != nil {
		log.Print("ERROR uploading data: " + rerr.Error())
		return
	}

	if resp.StatusCode != 200 {
		log.Printf("ERROR http response status code != 200: %d", resp.StatusCode)
	}

	// Consume POST response
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if len(body) != 0 {
		log.Printf("server response: %s\n", body)
	}
}
