package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"gopkg.in/fsnotify.v1"
)

const (
	packageManagerFolder = "MNEMO_PACKAGE_MANAGER_FOLDER"
	ansibleFilePath      = "MNEMO_ANSIBLE_FILE"
)

type Node struct {
	PackageName  string
	Dependencies []string
}

func main() {
	configPool, err := configLookup(packageManagerFolder, ansibleFilePath)
	if err != nil {
		panic(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	if err = watcher.Add(configPool[packageManagerFolder]); err != nil {
		panic(err)
	}

	for {
		select {
		case ev := <-watcher.Events:
			log.Println(ev)
			if ev.Op == fsnotify.Create || ev.Op == fsnotify.Remove {
				var nodes []Node
				if err = parsePackageTree(configPool[packageManagerFolder], &nodes); err != nil {
					panic(err)
				}

				nodesToInstall := evaluateNodesToInstall(&nodes)

				if err = writeFinalAnsibleFile(configPool, buildFile(&nodesToInstall)); err != nil {
					panic(err)
				}
			}
		case evErr := <-watcher.Errors:
			log.Println(evErr)
		}
	}

}

func writeFinalAnsibleFile(configPool map[string]string, fileContent []byte) error {
	f := []byte("- hosts: localhost\n  become: true\n  tasks:\n")
	f = append(f, fileContent...)
	return ioutil.WriteFile(configPool[ansibleFilePath], f, 0644)
}

func buildFile(nodesToInstall *[]string) (fileContent []byte) {
	for i := range *nodesToInstall {
		fileContent = append(fileContent, []byte("  - name: "+(*nodesToInstall)[i]+"\n    pacman: name="+(*nodesToInstall)[i]+"\n")...)
	}
	return
}

func configLookup(envs ...string) (map[string]string, error) {
	envsLookupTable := make(map[string]string)
	for i := range envs {
		env, ok := os.LookupEnv(envs[i])
		if !ok {
			return nil, errors.New(envs[i] + " not found")
		}
		envsLookupTable[envs[i]] = env
	}
	return envsLookupTable, nil
}

func evaluateNodesToInstall(nodes *[]Node) []string {
	var toInstall []string
	for i := range *nodes {
		if isCoveredDependency(nodes, (*nodes)[i].PackageName) {
			continue
		}
		toInstall = append(toInstall, (*nodes)[i].PackageName)
	}
	return toInstall
}

func isCoveredDependency(nodes *[]Node, packageName string) bool {
	for i := range *nodes {
		if contains(&(*nodes)[i].Dependencies, packageName) {
			return true
		}
	}
	return false
}

func contains(sarr *[]string, s string) bool {
	for i := range *sarr {
		if s == (*sarr)[i] {
			return true
		}
	}
	return false
}

func parsePackageTree(folder string, nodes *[]Node) (err error) {
	files, err := os.ReadDir(folder)
	if err != nil {
		return err
	}
	for _, v := range files {
		fullPath := folder + "/" + v.Name()
		if v.IsDir() {
			if err = parsePackageTree(fullPath, nodes); err != nil {
				return err
			}
		} else if v.Name() == "desc" {
			node, err := parseFile(fullPath)
			if err != nil {
				return err
			}
			*nodes = append(*nodes, node)
		}
	}
	return
}

func parseFile(filePath string) (Node, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Node{}, err
	}
	defer file.Close()
	var fileContent = make([]byte, getFileStat(file).Size())
	if _, err = file.Read(fileContent); err != nil {
		return Node{}, err
	}
	var node = new(Node)
	node.PackageName = extractName(fileContent)
	node.Dependencies = extractDependencies(fileContent)
	return *node, nil
}

func yankSection(f []byte, sectionTag string) []string {
	splitSection := strings.Split(string(f), sectionTag+"\n")
	if len(splitSection) == 1 {
		return []string{}
	}
	separatedLinesAfterSectionTag := strings.Split(splitSection[1], "\n")
	var yankSection []string
	for _, line := range separatedLinesAfterSectionTag {
		if len(line) == 0 || isASectionTag(line) {
			break
		}
		yankSection = append(yankSection, line)
	}
	return yankSection
}

func extractName(fileContent []byte) string {
	return yankSection(fileContent, "%NAME%")[0]
}

func extractDependencies(fileContent []byte) []string {
	sectionYank := yankSection(fileContent, "%DEPENDS%")
	if len(sectionYank) == 0 {
		return nil
	}

	for indexYank := range sectionYank {
		versionSeparator, has := containsStringSet(sectionYank[indexYank], "<", ">", "=")
		if !has {
			continue
		}
		sectionYank[indexYank] = strings.Split(sectionYank[indexYank], versionSeparator)[0]
	}
	return sectionYank
}

func containsStringSet(s string, sset ...string) (string, bool) {
	for _, v := range sset {
		if strings.Contains(s, v) {
			return v, true
		}
	}
	return "", false
}

func isASectionTag(s string) bool {
	return []rune(s)[0] == '%' && []rune(s)[len(s)-1] == '%'
}

func getFileStat(f *os.File) os.FileInfo {
	fileInfo, _ := f.Stat()
	return fileInfo
}
