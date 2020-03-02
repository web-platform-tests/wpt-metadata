## wpt-metadata
wpt-metadata is a repo for storing [wpt.fyi](https://github.com/web-platform-tests/wpt.fyi) test-result metadata, about tests defined in the [wpt](https://github.com/web-platform-tests/wpt) repo, in metadata yml files.

[Design Doc](https://docs.google.com/document/d/1oWYVkc2ztANCGUxwNVTQHlWV32zq6Ifq9jkkbYNbSAg/edit)

YAML Link Format Specs:

```
links:
  - product: [product spec] (optional)
    url: [URL]	
    results:
    - test: [Test name] 
      subtest: [Subtest name] (optional)
      status: [Specific test result status] (optional)
  - ...
```
    
Where
- `product` is a browser name with an option of [product spec](https://github.com/web-platform-tests/wpt.fyi/blob/master/api/README.md)
  - `{browser-name}[-{browser-version}[-{os}[-{os-version}]]]`
  - e.g. `chrome`, `safari-12`, or `firefox-63.0-linux`
- If omitted, the link will apply to all products in the directory.
- 'Test name' is a filename, which is relative to the current directory. If it
  is `"*"` (note that asterisks must be quoted in YAML), the link will apply to
  all tests in the current directory and its subdirectories.
- Test result status is a status as defined in the wpt.fyi codebase.

## How to contribute to wpt-metadata repository
You can contribute to the wpt-metadata repo by sending out a PR directly or through the Triage Metadata API available for trusted third parties.

### Triage Metadata API for trusted third parties
To use the Triage Metadata API, you first need to sign in to [wpt.fyi](https://wpt.fyi/) (top-right corner; 'Sign in with GitHub'). For more information on wpt.fyi login, see [here](https://docs.google.com/document/d/1iRkaK6cGgXp3DKbNbPMVsYGMaOHO-5CfqEuLPUR_2HM).

The logged-in user also needs to belong to the ['web-platform-tests' GitHub organization](https://github.com/orgs/web-platform-tests/people). To join, please [file an issue](https://github.com/web-platform-tests/wpt/issues/new?), including the reason you need access to the Triage Metadata API.

Once logged in, you can send a request to https://wpt.fyi/api/metadata/triage to triage metadata. This endpoint only accepts PATCH requests, with a body that contains a valid triage JSON object. The JSON object is a flattened YAML `Links` structure that is keyed by test name [Test path](https://docs.google.com/document/d/1oWYVkc2ztANCGUxwNVTQHlWV32zq6Ifq9jkkbYNbSAg/edit#heading=h.t7ysbpr8er1y); see below for an example.

`/api/metadata/triage` returns the url of a PR that is created in the wpt-metadata repo.

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
          "status": 6
        },
        {
          "subtest": "Invalid contentType (\"text/plain\")",
          "status": 0
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
