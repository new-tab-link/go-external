package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/gobwas/glob"
)

type Command struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}
type CommandConfig struct {
	Page    []string `json:"page"`
	Link    []string `json:"link"`
	Command Command  `json:"command"`
}

type Config struct {
	Port     string          `json:"port"`
	Commands []CommandConfig `json:"commands"`
}

type HttpBody struct {
	Page  string `json:"page"`
	Link  string `json:"link"`
	Rule  string `json:"rule"`
	Type  string `json:"type"`
	Extra string `json:"extra"`
}

func main() {
	run()
}

func run() {
	config, err := loadConfig()
	if err != nil {
		panic(err)
	}
	fmt.Println(config)
	err = listen(config)
	if err != nil {
		panic(err)
	}
}

func loadConfig() (*Config, error) {
	var fn string
	switch runtime.GOOS {
	case "windows":
		fn = "config.windows.json"
	case "linux":
		fn = "config.linux.json"
	case "darwin":
		fn = "config.darwin.json"
	default:
		return nil, errors.New("unknown OS")
	}
	bs, err := loadFile(fn)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = json.Unmarshal(bs, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func loadFile(fn string) ([]byte, error) {
	fi, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	bs, err := io.ReadAll(fi)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func testUrl(globs []string, url string) bool {
	for _, g := range globs {
		if glob.MustCompile(g).Match(url) {
			return true
		}
	}
	return false
}

func parseCommand(body *HttpBody, commands []CommandConfig) error {
	for _, command := range commands {
		if !testUrl(command.Page, body.Page) || !testUrl(command.Link, body.Link) {
			continue
		}
		runCommand(command.Command, body.Link)
		return nil
	}

	return nil
}

func runCommand(command Command, link string) error {
	err := exec.Command(command.Name, append(command.Args, link)...).Run()
	if err != nil {
		fmt.Println(err)
	}
	return nil
}
func listen(config *Config) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var body HttpBody
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		w.WriteHeader(http.StatusOK)
		go parseCommand(&body, config.Commands)
	})
	err := http.ListenAndServe(":"+config.Port, nil)
	if err != nil {
		return err
	}
	return nil
}
