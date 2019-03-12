package util

import (
	"fmt"
	"os"
	"path/filepath"	
	"io/ioutil"
	"strings"
)

func CheckError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

// CollectMetadata iterates through wpt-metadata repository and returns a
// map that maps a test path to its META.yml file content.
func CollectMetadata() map[string][]byte{
	ymlFiles := []string{}
	curPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	rootDor := filepath.Dir(filepath.Dir(curPath))
	walkErr := filepath.Walk(rootDor, func(path string, f os.FileInfo, err error) error {
		if f.Name() == "META.yml" {
			ymlFiles = append(ymlFiles, path)
		}	
        return err
	})
	CheckError(walkErr)

	ymlMap := make(map[string][]byte)
	for _, file := range ymlFiles {
		data, err := ioutil.ReadFile(file)
		CheckError(err)
		testPath := strings.TrimPrefix(file, rootDor)
		ymlMap[testPath] = data 
	}
	return ymlMap
}