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
- Test path is a filename, which is relative to the current directory. If it is `*`, the link will apply to all tests in the current directory and its subdirectories.
- Test result status is a status as defined in the wpt.fyi codebase.
