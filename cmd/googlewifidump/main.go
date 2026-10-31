package main

import (
	"encoding/json"
	"flag"
	"github.com/bengalaviz/googlewifi/diagnosticreport"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type loadMode int

const (
	viaHTTP loadMode = iota
	viaFile
)

func main() {
	flag.Parse()
	source := flag.Arg(0)
	if source == "" {
		source = "http://192.168.86.1/api/v1/diagnostic-report"
	}
	var mode loadMode
	if uri, err := url.ParseRequestURI(source); err != nil || uri.Scheme == "" {
		mode = viaFile
	} else {
		mode = viaHTTP
	}
	var reader io.Reader
	switch mode {
	case viaHTTP:
		data, contentType, resolvedSource, err := fetchReport(source)
		if err != nil {
			log.Fatalln("error getting report:", err)
		}
		if strings.Contains(contentType, "application/json") {
			var value interface{}
			if err := json.Unmarshal(data, &value); err != nil {
				log.Fatalln("error parsing JSON from", resolvedSource, ":", err)
			}
			if err := json.NewEncoder(os.Stdout).Encode(value); err != nil {
				log.Fatalln("error encoding JSON:", err)
			}
			return
		}
		reader = strings.NewReader(string(data))
	case viaFile:
		file, err := os.Open(source)
		if err != nil {
			log.Fatalln("error opening report:", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Fatalln("error closing opened report:", err)
			}
		}()
		reader = file
	default:
		log.Fatalln("bad source mode")
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatalln("error reading report:", err)
	}

	dr, err := diagnosticreport.Parse(data)
	if err != nil {
		log.Fatalln("error parsing report:", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(dr); err != nil {
		log.Fatalln("error encoding report:", err)
	}
}

func statusFallbackURL(source string) (string, error) {
	u, err := url.Parse(source)
	if err != nil {
		return "", err
	}
	u.Path = "/api/v1/status"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func fetchReport(source string) ([]byte, string, string, error) {
	resp, err := http.Get(source)
	if err != nil {
		return nil, "", source, err
	}
	defer resp.Body.Close()
	data, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, "", source, readErr
	}
	if resp.StatusCode == http.StatusNotFound {
		fallback, err := statusFallbackURL(source)
		if err != nil {
			return nil, "", source, err
		}
		fallbackResp, err := http.Get(fallback)
		if err != nil {
			return nil, "", fallback, err
		}
		defer fallbackResp.Body.Close()
		fallbackData, err := ioutil.ReadAll(fallbackResp.Body)
		if err != nil {
			return nil, "", fallback, err
		}
		if fallbackResp.StatusCode < 200 || fallbackResp.StatusCode >= 300 {
			return nil, "", fallback, &httpError{status: fallbackResp.StatusCode}
		}
		return fallbackData, fallbackResp.Header.Get("Content-Type"), fallback, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", source, &httpError{status: resp.StatusCode}
	}
	return data, resp.Header.Get("Content-Type"), source, nil
}

type httpError struct{ status int }

func (e *httpError) Error() string { return "HTTP status " + http.StatusText(e.status) }
