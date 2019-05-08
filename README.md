Repo for storing wpt meatadata yml files.
[Design Doc](https://docs.google.com/document/d/1oWYVkc2ztANCGUxwNVTQHlWV32zq6Ifq9jkkbYNbSAg/edit)

YAML Link Format Specs:

```
links:
  - product: [browser name/product spec] (optional)
    test: [Test path]
    status: [Test result status] (optional)
    url: [URL]	
  - ...
```
    
Where
- `product` is a browser name with an option of [product spec](https://github.com/web-platform-tests/wpt.fyi/blob/master/api/README.md)
  - `{browser-name}[-{browser-version}[-{os}[-{os-version}]]]`
  - e.g. `chrome`, `safari-12`, or `firefox-63.0-linux`
- If omitted, the link will apply to all products in the directory.
- Test paths are relative to the current directory, so will typically be just a filename.
- Test result status is a status as defined in the wpt.fyi codebase.
