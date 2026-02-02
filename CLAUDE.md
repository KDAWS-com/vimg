vimg is based off of an earlier version of https://github.com/h2non/bimg/

We made changes because bimg passed around large objects from one operation to another, causing memory copies and slowdowns, whereas we moved to a fluent interface to chain operations on the same VIPS image resource.  Latency is key.

## Testing

To run tests, set the TMPDIR environment variable to a writable location:

```bash
export TMPDIR=~/tmp
mkdir -p ~/tmp
go test ./...
```
