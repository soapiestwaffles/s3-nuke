[![Lint & Test](https://github.com/soapiestwaffles/s3-nuke/actions/workflows/lint.yml/badge.svg)](https://github.com/soapiestwaffles/s3-nuke/actions/workflows/lint.yml) [![Coverage Status](https://coveralls.io/repos/github/soapiestwaffles/s3-nuke/badge.svg?branch=main)](https://coveralls.io/github/soapiestwaffles/s3-nuke?branch=main) [![Go Report Card](https://goreportcard.com/badge/github.com/soapiestwaffles/s3-nuke)](https://goreportcard.com/report/github.com/soapiestwaffles/s3-nuke)

⚠️ *Work In Progress - this project is still in early stages and is currently non-functional* ⚠️

![header image](https://github.com/soapiestwaffles/_assets/raw/master/s3-nuke/header.jpg)

# 🪣💣 s3-nuke 💣🪣
Nuke all the files and their versions from an S3 Bucket 

### Why?
_...because deleting any bucket with files is just plain annoying._

While there are several other scripts and projects that do the same function, I wanted something
with a little more interactivity. (Also, I was bored 😊)

### Required AWS policy

TODO

## Misc Tools

### s3-metrics

This tool will return back the current and historical approximate number of objects in a bucket using CloudWatch.

#### Installing s3-metrics binary

* using `go`:
```
go install github.com/soapiestwaffles/s3-nuke/tools/s3-metrics@latest
```

#### Running s3-metrics from source
```
go run tools/s3-metrics/main.go
```

#### Usage/Available flags

```
s3-metrics tool: get bucket object metrics for a particular bucket

Flags:
  -h, --help                   Show context-sensitive help.
  -e, --aws-endpoint=STRING    override AWS endpoint address ($AWS_ENDPOINT)
  -r, --region="us-west-2"     specify region to create bucket and objects in ($AWS_REGION)
```

#### Example output
```console
$ go run tools/s3-metrics/main.go

🪣  my-s3-bucket

--> bucket located in us-east-1

 544015 ┤                                         ╭─────────────────
 544004 ┤                                       ╭─╯
 543994 ┤                                     ╭─╯
 543983 ┤                                    ╭╯
 543973 ┤                                   ╭╯
 543962 ┤                                  ╭╯
 543951 ┤                                 ╭╯
 543941 ┤                                ╭╯
 543930 ┤                           ╭────╯
 543920 ┤              ╭────────────╯
 543909 ┼──────────────╯
                        Object Count for past 30 Days

Approx. objects currently in bucket: 544015
Metric last updated: 2022-01-17 00:00:00 +0000 UTC
```

### s3-gen

This tool was created mainly for testing s3-nuke. This tool will generate `num-buckets` number of buckets, each containing `num-objects` number of objects (containing random data), with `num-versions` number of versions. If `num-versions` < 2, s3-gen will create the buckets with versioning disabled.

#### Installing s3-metrics binary

* using `go`:
```
go install github.com/soapiestwaffles/s3-nuke/tools/s3-gen@latest
```

#### Running s3-gen from source

```
go run tools/s3-gen/main.go --num-buckets=INT --num-objects=INT --num-versions=INT
```

#### Usage/Available flags

```
s3-nuke tool: generate a set of randomized buckets each containing a set of randomized objects and versions

Flags:
  -h, --help                     Show context-sensitive help.
  -e, --aws-endpoint=STRING      override AWS endpoint address ($AWS_ENDPOINT)
  -n, --num-buckets=INT          number of buckets with randomized names to create
  -o, --num-objects=INT          number of random objects generated and put into buckets
  -v, --num-versions=INT         number of versions to create for each random object
  -r, --region="us-west-2"       specify region to create bucket and objects in
  -p, --bucket-prefix="s3gen"    prefix for every bucket name generated
      --yes                      bypass user prompt and proceed with action automatically
```

## License

MIT License

Copyright (c) 2021 SoapiestWaffles

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
