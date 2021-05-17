package main

import (
    "bufio"
    "fmt"
    mapset "github.com/deckarep/golang-set"
    "github.com/web-platform-tests/wpt.fyi/shared"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "log"
    "os"
    "path"
    "strings"
)

func main() {
    portTestExpectations()
}

func portTestExpectations() {
    //f, err := os.Open("TestExpectations.txt")
    //f, err := os.Open("PortableTestExpectation.txt")
    f, err := os.Open("NeverFixTests")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    //bsfSet := getBSF()
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        expectation := scanner.Text()
        url, tags, testname, results := parseExpectation(expectation)

        if isMeetNeverFixCriteria(url, tags, testname, results) {
            fmt.Println(expectation)
            //if bsfSet.Contains(testname) {
            //}
        }

        //toWPTMetadata(url, testname, results)

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

func isMeetNeverFixCriteria(url, tags, testname, results string) bool {
    if url == "" {
        return false
    }

    if testname == "" {
        return false
    }

    if results != "Skip" {
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
            testname = split[i][len("external/wpt/"):]
            //fmt.Println(testname)
        }
    }

    return
}

func toWPTMetadata(url, filename, expectationStatus string) {
    dirpath := path.Dir(filename)
    testpath := path.Base(filename)
    createDirIfNeeded(dirpath)
    status := toMetadataStatus(expectationStatus)

    filepath := dirpath + "/META.yml"
    f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0755)
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    fi, err := f.Stat()
    if err != nil {
        log.Fatal(err)
    }

    var metadata shared.Metadata
    if fi.Size() == 0 {
        fmt.Println("New File....")
        // New metadata
        metadata = shared.Metadata{
            Links: []shared.MetadataLink{
                shared.MetadataLink{
                    Product: shared.ParseProductSpecUnsafe("chrome"),
                    URL:     url,
                    Results: []shared.MetadataTestResult{{
                        TestPath: testpath,
                        Status:   &status,
                    }},
                },
            },
        }
        writeToFile(metadata, f)
        return
    }

    data, err := ioutil.ReadAll(f)
    if err != nil {
        log.Fatal(err)
    }

    hasAdded := false
    err = yaml.Unmarshal(data, &metadata)
    for index, link := range metadata.Links {
        if link.Product.String() != "chrome" {
            continue
        }

        for _, result := range link.Results {
            if result.TestPath == testpath {
                return
            }
        }

        // Add a new result under the same URL
        if link.URL == url {
            fmt.Println("New Result...")
            metadata.Links[index].Results = append(link.Results, shared.MetadataTestResult{TestPath: testpath, Status: &status})
            hasAdded = true
            break
        }
    }

    // New link
    if !hasAdded {
        fmt.Println("New Link...")
        newLink := shared.MetadataLink{
            Product: shared.ParseProductSpecUnsafe("chrome"),
            URL:     url,
            Results: []shared.MetadataTestResult{{
                TestPath: testpath,
                Status:   &status,
            }},
        }
        metadata.Links = append(metadata.Links, newLink)
    }

    writeToFile(metadata, f)
}

func writeToFile(metadata shared.Metadata, f *os.File) {
    fmt.Println("Write to file...")
    metadataBytes, err := yaml.Marshal(metadata)
    if err != nil {
        log.Fatal(err)
    }

    _, err = f.WriteAt(metadataBytes, 0)
    if err != nil {
        log.Fatal(err)
    }
}

func toMetadataStatus(status string) shared.TestStatus {
    if status == "Failure" {
        return shared.TestStatusFail
    }

    return shared.TestStatusTimeout
}

func createDirIfNeeded(dir string) {
    _, err := os.Stat(dir)
    if !os.IsNotExist(err) {
        return
    }

    fmt.Println("making dir")
    os.MkdirAll(dir, os.ModePerm)
}

func getBSF() mapset.Set {
    f, err := os.Open("BSF.txt")
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

// Parser for getting all Chrome-specific test failures.
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

func processMonorailFruits() {
    peSet := getPESet()
    // Read monorail-CSF-results.txt
    f, err := os.Open("monorail-CSF-results.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        monorailResult := scanner.Text()
        // Get tests with one link
        testname, bug := parseMonorailResult(monorailResult)
        if testname == "" {
            continue
        }

        // Filter out the ported Test Expectations.
        if peSet.Contains(testname[1:]) {
            continue
        }

        toWPTMetadata("crbug.com/"+bug, testname[1:], "Failure")
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

}

func parseMonorailResult(monorailResult string) (string, string) {
    split := strings.Split(monorailResult, "=>")
    testname := strings.TrimSpace(split[0])
    right := strings.TrimSpace(split[1])
    if right == "[]" {
        return "", ""
    }

    bugs := strings.Split(right, ",")
    if len(bugs) != 1 {
        return "", ""
    }

    bug := bugs[0][1 : len(bugs[0])-1]
    return testname, bug
}

func getPESet() mapset.Set {
    // A non-filtered list of portable Test Expectations.
    f, err := os.Open("PE.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    peSet := mapset.NewSet()
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        expectation := scanner.Text()
        _, _, testname, _ := parseExpectation(expectation)
        peSet.Add(testname)
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

    return peSet
}

func getOptOutSet() mapset.Set {
    // A non-filtered list of portable Test Expectations.
    f, err := os.Open("PE.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    peSet := mapset.NewSet()
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        expectation := scanner.Text()
        _, _, testname, _ := parseExpectation(expectation)
        peSet.Add(testname)
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

    return peSet
}
