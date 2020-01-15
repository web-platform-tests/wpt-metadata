## wpt-metadata
wpt-metadata is a repo for storing [wpt.fyi](https://github.com/web-platform-tests/wpt.fyi) test-result metadata, about tests defined in the [wpt](https://github.com/web-platform-tests/wpt) repo, in metadata yml files.

[Design Doc](https://docs.google.com/document/d/1oWYVkc2ztANCGUxwNVTQHlWV32zq6Ifq9jkkbYNbSAg/edit)

YAML Link Format Specs:

```
links:
  - product: [product spec] (optional)
    url: [URL]	
    results:
    - test: [Test path] 
      subtest: [Subtest name] (optional)
      status: [Specific test result status] (optional)
  - ...
```
    
Where
- `product` is a browser name with an option of [product spec](https://github.com/web-platform-tests/wpt.fyi/blob/master/api/README.md)
  - `{browser-name}[-{browser-version}[-{os}[-{os-version}]]]`
  - e.g. `chrome`, `safari-12`, or `firefox-63.0-linux`
- If omitted, the link will apply to all products in the directory.
- Test path is relative to the current directory, so will typically be just a filename.
- Test result status is a status as defined in the wpt.fyi codebase.

## How to contribute to wpt-metadata repository
If you are a memeber of the web-platform-tests org, you can contribute to the wpt-metadata repo by sending out a PR directly or through a UI-based approach on wpt.fyi. We also have a Triage Metadata API available for trusted third parties

### Triage through wpt.fyi UI
For information about how to use the triage service on wpt.fyi, see here - UPCOMING

### Triage Metadata API for trusted third parties
In order to use the Triage Metadata API, first you need to log in on [wpt.fyi](https://wpt.fyi/) through the GitHub OAuth integration. For more information on wpt.fyi login, see [here](https://docs.google.com/document/d/1iRkaK6cGgXp3DKbNbPMVsYGMaOHO-5CfqEuLPUR_2HM)

The logged-in user also needs to be a part of the web-platform-tests org. To join, please file an issue or contact us directly.

Upon a login, you can send a request to /api/metadata/triage endpoint to triage metadata. This endpoint only accepts PATCH request and JSON object body. The JSON object is a flattened YAML `Links` structure which is keyed by test name [Test path].

/api/metadata/triage returns the url of a PR that is created in the wpt-metadata repo.

<details><summary><b>Example JSON Body</b></summary>

```json
{
  "/FileAPI/blob/Blob-constructor.html": [
    {
      "url": "https://github.com/web-platform-tests/results-collection/issues/661",
      "product": "chrome",
      "results:" [
        {
          "subtest": "Blob with type \"image/gif;\"",
          "status": "UNKNOWN"
        },
        {
          "subtest": "Invalid contentType (\"text/plain\")",
          "status": "UNKNOWN"
        }
      ]
    }
  ],
  "/service-workers/service-worker/fetch-request-css-base-url.https.html": [
    {
      "url": "https://bugzilla.mozilla.org/show_bug.cgi?id=1201160",
      "product": "firefox",
    }
  ],
  "/service-workers/service-worker/fetch-request-css-images.https.html": [
    {
      "url": "https://bugzilla.mozilla.org/show_bug.cgi?id=1532331",
      "product": "firefox"
    }
  ]
}
```
