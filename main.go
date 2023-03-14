package main

import (
	"fmt"
	"os"
	"strings"
)

type Node struct {
	PackageName  string
	Dependencies []string
}

func main() {
	var nodes []Node
	err := parsePackageTree("/var/lib/pacman/local", &nodes)
	if err != nil {
		panic(err)
	}
	fmt.Println(nodes[0])
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

func extractName(fileContent []byte) string {
	return yankSection(fileContent, "%NAME%")[0]
}

func extractDependencies(fileContent []byte) []string {
	return yankSection(fileContent, "%DEPENDS%")
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

func isASectionTag(s string) bool {
	return []rune(s)[0] == '%' && []rune(s)[len(s)-1] == '%'
}

func getFileStat(f *os.File) os.FileInfo {
	fileInfo, _ := f.Stat()
	return fileInfo
}
