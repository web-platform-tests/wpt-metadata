package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// CollectMetadata iterates through wpt-metadata repository and returns a
// map that maps a test path to its META.yml file content.
func CollectMetadata() map[string][]byte {
	ymlFiles := []string{}
	curPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	rootDir := filepath.Dir(filepath.Dir(curPath))
	walkErr := filepath.Walk(rootDir, func(path string, f os.FileInfo, err error) error {
		if f.Name() == "META.yml" {
			ymlFiles = append(ymlFiles, path)
		}
		return err
	})
	if walkErr != nil {
		fmt.Println(walkErr)
	}

	ymlMap := make(map[string][]byte)
	for _, file := range ymlFiles {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Println(err)
		}
		testPath := strings.TrimPrefix(file, rootDir)
		ymlMap[testPath] = data
	}

	return ymlMap
}
