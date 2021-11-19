## wpt-metadata
wpt-metadata is a repo for storing [wpt.fyi](https://github.com/web-platform-tests/wpt.fyi) test-result metadata, about tests defined in the [wpt](https://github.com/web-platform-tests/wpt) repo, in metadata yml files.

[Design Doc](https://docs.google.com/document/d/1oWYVkc2ztANCGUxwNVTQHlWV32zq6Ifq9jkkbYNbSAg/edit)

YAML Link Format Specs:

```
links:
  - product: [product spec] (optional)
    url: [URL]
    label: [Label]
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
  - when `product` is omitted, this `link` applies to tests across all browsers
- `url` is a bug URL; `url` is non-optional unless a `label` is present.
- `label` is a label to its tests; `label` is used at a test-level and do not
  apply to subtests. When `label` is present in a link:
  - `product`should be omitted
  - `url` is optional
  - `Subtest name` is omitted
- `Test name` is a filename, which is relative to the current directory. If it
  is `"*"` (note that asterisks must be quoted in YAML), the link will apply to
  all tests in the current directory and its subdirectories.
- `Test result status` is an optional field that records the WPT test result, as 
  [defined](https://github.com/web-platform-tests/wpt.fyi/blob/master/shared/statuses.go#L52) 
  in the wpt.fyi codebase. When the WPT version or the browser version 
  changes, this field could be used to indicate that a test is out-of-date. It is currently unused by tooling.

## How to contribute to wpt-metadata repository
You can contribute to the wpt-metadata repo by sending out a PR directly or through the Triage Metadata API available for trusted third parties.

### Triage Metadata API for trusted third parties
See [/api/metadat/triage](https://github.com/web-platform-tests/wpt.fyi/tree/master/api#apimetadatatriage) for more information.
