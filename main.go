package main

import (
	"bufio"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {
	start()
}

func start() {
	f, err := os.Open("TestExpectations.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		expectation := scanner.Text()
		url, tags, testname, results := parseExpectation(expectation)
		if isMeetCriteria(url, tags, testname, results) {
			bsfSet := getBSF()
			if bsfSet.Contains(testname) {
				fmt.Println(expectation)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

// https://github.com/web-platform-tests/wpt.fyi/issues/1993#issuecomment-654551961
func isMeetCriteria(url, tags, testname, results string) bool {
	if url == "" {
		return false
	}

	if testname == "" {
		return false
	}

	if results != "Timeout" && results != "Failure" {
		return false
	}

	if tags != "" {
		if !strings.Contains(tags, "Linux") {
			if tags != "Release" {
				return false
			}
		} else {
			if strings.Contains(tags, "Debug") {
				return false
			}
		}
	}
	return true
}

// syntax: https://chromium.googlesource.com/chromium/src/+/master/docs/testing/web_test_expectations.md#syntax
// [ bugs ] [ "[" modifiers "]" ] test_name [ "[" expectations "]" ]
func parseExpectation(line string) (url string, tags string, testname string, results string) {
	if strings.HasPrefix(line, "# ") {
		return
	}

	split := strings.Split(line, " ")
	//fmt.Println("new line")
	//fmt.Println(split)
	for i := 0; i < len(split); i++ {
		if strings.HasPrefix(split[i], "crbug.com") {
			url = split[i]
			//fmt.Println(url)
		}

		if split[i] == "[" {
			val := ""
			i++
			for split[i] != "]" {
				val += split[i]
				i++
			}

			if testname == "" {
				tags = val
				//fmt.Println(tags)
			} else {
				results = val
				//fmt.Println(results)
			}
		}

		if strings.HasPrefix(split[i], "external/wpt/") {
			//fmt.Println(split[i])
			testname = split[i][len("external/wpt"):]
			//fmt.Println(testname)
		}
	}

	return
}

func toWPTMetadata(line string) string {
	return ""
}

func getBSF() mapset.Set {
	f, err := os.Open("portableBSF.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	bsfSet := mapset.NewSet()
	for scanner.Scan() {
		bsf := scanner.Text()
		bsfSet.Add(bsf)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return bsfSet
}

func parseBSF() {
	data, err := ioutil.ReadFile("BSF.txt")
	if err != nil {
		fmt.Println("File reading error", err)
		return
	}
	bsf := string(data)
	vals := strings.Split(bsf, ",")
	for _, part := range vals {
		if strings.HasPrefix(part, "{\"test\":") {
			fmt.Println(part[len("{\"test\":"):])
		}
	}
}
