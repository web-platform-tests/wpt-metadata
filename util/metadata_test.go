package util

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCollectMetadataFromURL_Success(t *testing.T) {
	linkNode := []byte(`
links:
- product: chrome-64
test: a.html
url: https://external.com/item
- product: firefox-2
test: b.html
url: https://bug.com/item`)

	fileName := "META.yml"
	createFile(fileName, linkNode)

	file, err := os.Create("output.tar.gz")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()
	// set up the gzip writer
	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	// grab the paths that need to be added in
	paths := []string{
		"META.yml",
	}
	// add each file as needed into the current tar archive
	for i := range paths {
		if err := addFile(tw, paths[i]); err != nil {
			log.Fatalln(err)
		}
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, _ := os.Open("output.tar.gz")
		reader, _ := gzip.NewReader(f)
		fmt.Fprint(w, reader)

	}))
	defer ts.Close()
	metadataMap, err := collectMetadataFromURL(http.DefaultClient, ts.URL)
	assert.Nil(t, err)
	fmt.Println(metadataMap)
}

func createFile(fileName string, content []byte) {
	f, _ := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	f.Write(content)

	f.Close()
}

func addFile(tw *tar.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if stat, err := file.Stat(); err == nil {
		// now lets create the header as needed for this file within the tarball
		header := new(tar.Header)
		header.Name = path
		header.Size = stat.Size()
		header.Mode = int64(stat.Mode())
		header.ModTime = stat.ModTime()
		// write the header to the tarball archive
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// copy the file data to the tarball
		if _, err := io.Copy(tw, file); err != nil {
			return err
		}
	}
	return nil
}
