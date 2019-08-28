package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

func main() {
	flagHelp := flag.Bool("help", false, "print this help")
	flagDepGopkgLockPath := flag.String("d", "", "dep Gopkg.lock file")
	flagGoListModAllPath := flag.String("m", "", "file containing the output of 'go list -m all'")
	flagGitHubAuthUsername := flag.String("u", "", "username to auth with GitHub (optional)")
	flagGitHubAuthPassword := flag.String("p", "", "password or personal access token to auth with GitHub (optional)")

	flag.Parse()

	if *flagHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *flagDepGopkgLockPath == "" {
		fmt.Fprintln(os.Stderr, "No dep Gopkg.lock file provided.")
		os.Exit(1)
	}

	if *flagGoListModAllPath == "" {
		fmt.Fprintln(os.Stderr, "No file containing the output of 'go list -m all' provided.")
		os.Exit(1)
	}

	err := run(*flagDepGopkgLockPath, *flagGoListModAllPath, *flagGitHubAuthUsername, *flagGitHubAuthPassword)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(depGopkgLockPath, goListModAllPath, githubAuthUsername, githubAuthPassword string) error {
	depGopkgLockDeps, err := depGopkgLockToMap(depGopkgLockPath)
	if err != nil {
		return err
	}

	goListModAllDeps, err := goListModAllToMap(goListModAllPath)
	if err != nil {
		return err
	}

	importPaths := getAllKeysSorted(depGopkgLockDeps, goListModAllDeps)
	fmt.Println("Removed:")
	for _, ip := range importPaths {
		dep := depGopkgLockDeps[ip]
		mod := goListModAllDeps[ip]
		if len(dep) > 0 && len(mod) == 0 {
			fmt.Println("- ", ip, dep)
		}
	}
	fmt.Println()
	fmt.Println("Added:")
	for _, ip := range importPaths {
		dep := depGopkgLockDeps[ip]
		mod := goListModAllDeps[ip]
		if len(dep) == 0 && len(mod) > 0 {
			fmt.Println("+ ", ip, mod)
		}
	}
	fmt.Println()
	fmt.Println("Changed:")
	for _, ip := range importPaths {
		dep := depGopkgLockDeps[ip]
		mod := goListModAllDeps[ip]
		if len(dep) > 0 && len(mod) > 0 && dep != mod {
			tags, err := getTagsAndRevisions(ip, githubAuthUsername, githubAuthPassword)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error retrieving alternative tags:", err)
			} else if tags[dep] == tags[mod] || tags[dep] == mod || tags[mod] == dep {
				continue
			}
			fmt.Println("! ", ip, dep, "=>", mod)
		}
	}

	return nil
}

func depGopkgLockToMap(depGopkgLockPath string) (map[string]string, error) {
	depGopkgLockContents, err := ioutil.ReadFile(depGopkgLockPath)
	if err != nil {
		return nil, fmt.Errorf("error reading dep Gopkg.lock file: %w", err)
	}

	type depGopkgLockProject struct {
		Name     string
		Revision string
		Version  string
	}

	type depGopkgLockFile struct {
		Projects []depGopkgLockProject
	}

	depGopkgLock := depGopkgLockFile{}
	_, err = toml.Decode(string(depGopkgLockContents), &depGopkgLock)
	if err != nil {
		return nil, fmt.Errorf("error decoding dep Gopkg.lock file: %w", err)
	}

	depGopkgLockDeps := map[string]string{}
	for _, p := range depGopkgLock.Projects {
		version := ""
		if p.Version != "" {
			version = p.Version
		}
		if version == "" {
			version = p.Revision[:12]
		}
		depGopkgLockDeps[p.Name] = version
	}

	return depGopkgLockDeps, nil
}

func goListModAllToMap(goListModAllPath string) (map[string]string, error) {
	goListModAllContents, err := ioutil.ReadFile(goListModAllPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file containing the output of 'go list -m all': %w", err)
	}

	// TODO: Replace with consuming the 'json' output format of go list.

	goListModAllDeps := map[string]string{}
	for _, l := range strings.Split(string(goListModAllContents), "\n") {
		components := strings.Split(l, " ")
		if len(components) < 2 {
			continue
		}
		name := components[0]
		version := components[1]
		if strings.Contains(version, "+") {
			version = strings.Split(version, "+")[0]
		}
		if strings.Contains(version, "-") {
			version = strings.Split(version, "-")[2]
		}
		goListModAllDeps[name] = version
	}

	return goListModAllDeps, nil
}

func getAllKeysSorted(maps ...map[string]string) []string {
	imports := map[string]bool{}
	for _, m := range maps {
		for k := range m {
			imports[k] = true
		}
	}
	keys := make([]string, 0, len(imports))
	for k := range imports {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func getTagsAndRevisions(importPath, githubUsername, githubPassword string) (map[string]string, error) {
	if !strings.HasPrefix(importPath, "github.com/") {
		return nil, nil
	}

	repo := strings.TrimPrefix(importPath, "github.com/")

	url := "https://api.github.com/repos/" + repo + "/tags"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if githubUsername != "" && githubPassword != "" {
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(githubUsername+":"+githubPassword)))
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	type tag struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		}
	}

	var tags []tag
	err = json.NewDecoder(res.Body).Decode(&tags)
	if err != nil {
		return nil, err
	}

	tagMap := map[string]string{}
	for _, t := range tags {
		tagMap[t.Name] = t.Commit.SHA[:12]
	}
	return tagMap, nil
}
