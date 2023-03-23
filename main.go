package main

import (
	"fmt"
	"mnemo/gitwrapper"
	"os"
	"strings"
)

type Node struct {
	PackageName  string
	Dependencies []string
}

func main() {
	gitwrapper.GitInit()
	defer gitwrapper.GitShutdown()

	var nodes []Node
	err := parsePackageTree("/var/lib/pacman/local", &nodes)
	if err != nil {
		panic(err)
	}
	// shitty but works
	toInstall := evaluateNodesToInstall(&nodes)
	fmt.Println(toInstall)
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
