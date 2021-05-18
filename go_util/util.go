// Copyright 2019 The WPT Dashboard Project. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package go_util

import (
	"fmt"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

func AppendPendingMetadata(testName string, link shared.MetadataLink, metadata *shared.Metadata) {
	// First check whether we have an existing link in this META.yml; if
	// so we just want to append our testname to the set of tests associated
	// with it.
	hasAdded := false
	for index, existingLink := range metadata.Links {
		if existingLink.URL == link.URL && link.Product.MatchesProductSpec(existingLink.Product) {
			fmt.Println("APPEND-LINK: for test", testName, "with url", link.URL)
			metadata.Links[index].Results = append(existingLink.Results, link.Results...)
			hasAdded = true
			break
		}
	}

	// Otherwise, this is a brand new entry in the META.yml, so add a new
	// link to it.
	if !hasAdded {
		fmt.Println("NEW-LINK: for test", testName, "with url", link.URL)
		metadata.Links = append(metadata.Links, link)
	}
}
