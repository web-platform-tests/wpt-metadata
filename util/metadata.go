package util

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

const url = "https://api.github.com/repos/web-platform-tests/wpt-metadata/tarball"

// CollectMetadata iterates through wpt-metadata repository and returns a
// map that maps a test path to its META.yml file content.
func CollectMetadata() (res map[string][]byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	// The statuscode could be one of redirect codes.
	if !(statusCode == 200 || (statusCode >= 300 && statusCode <= 308)) {
		err := fmt.Errorf("Bad status code:%d, Unable to download wpt-metadata", statusCode)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	gzip, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer gzip.Close()

	tarReader := tar.NewReader(gzip)
	var metadataMap = make(map[string][]byte)
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		// Not a regular file.
		if header.Typeflag != tar.TypeReg {
			continue
		}

		if !strings.HasSuffix(header.Name, "META.yml") {
			continue
		}

		data := make([]byte, header.Size)

		_, err = tarReader.Read(data)
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Removes `owner-repo` prefix in the file name.
		relativeFileName := header.Name[strings.Index(header.Name, "/")+1:]
		metadataMap[relativeFileName] = data
	}

	return metadataMap, nil
}
