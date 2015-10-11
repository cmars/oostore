[![Build Status](https://travis-ci.org/cmars/oostore.svg?branch=master)](https://travis-ci.org/cmars/oostore)
[![GoDoc](https://godoc.org/github.com/cmars/oostore?status.svg)](https://godoc.org/github.com/cmars/oostore)

# oostore

An opaque object storage service.

oostore is designed for sharing small amounts of content. In exchange for
posted content, oostore responds with a macaroon that may be used to retrieve
or delete the content.

Caveats may then be added to this macaroon -- which can place all sorts of
conditions on its validity. Just about any kind of authorization policy
imaginable should be possible to implement, by adding caveats to opaque content
macaroons.

## Macaroons? What?
For more background on what macaroons are all about and why you might want to
use them, I recommend:
* [Macaroons are Better Than Cookies!](http://hackingdistributed.com/2014/05/16/macaroons-are-better-than-cookies/)
* [Macaroons: Cookies with Contextual Caveats for Decentralized Authorization in the Cloud](https://air.mozilla.org/macaroons-cookies-with-contextual-caveats-for-decentralized-authorization-in-the-cloud/)

First-party caveats currently understood by this service:

### object _object-id_
Request must operate on this object. oostore will currently add this caveat
automatically when a new object is created and a macaroon is issued, in response
so that the creator can manage it, and distribute authorization to others.

### time-before _RFC3339-timestamp_
Authorization expires after a set time. The timestamp is compared against
current time on the oostore server. This caveat is provided by the [macaroon-bakery](https://godoc.org/gopkg.in/macaroon-bakery.v1/bakery/checkers).

### client-ip-addr _w.x.y.z_
Only client requests from a specific IP address are allowed. This caveat is provided by the [macaroon-bakery](https://godoc.org/gopkg.in/macaroon-bakery.v1/bakery/checkers).

Stay tuned as oostore will become much more interesting (and useful, and
secure) once third-party caveats can be added against discharging services
designed for use with oostore.

# HTTP API
Macaroons are given in response to resource creation, and then sent as request
content for authorization. I feel like this is somewhat unorthodox for a web
API (usually you'd use cookies, and a 401 challenge-response) but for this use
case, it seemed appropriate. Plus, cookies have limitations on size, domains,
etc.

Support for authentication with cookies may be added on later to enable simple
web clients like browsers, etc. for specific use cases.

## POST /
Create a new object.

### Parameters
- [Header] Content-Type: _Will be stored with opaque object, preserved on retrieval. Defaults to application/octet-stream_
- [Contents] opaque object bytes

### Response 200 OK
- [Header] Location: _Path of newly created object._
- [Header] Content-Type: application/json
- [Contents] _The JSON-encoded macaroon, which is your authorization token for the object._

### Example
```
$ curl -i -X POST --data "good things" http://localhost:20080
HTTP/1.1 200 OK
Location: /7zCHWLjyMohzSrKUHRg2wLMb4hvPkV7mdEeDbweAhJZj
Date: Sat, 19 Sep 2015 04:31:52 GMT
Content-Length: 235
Content-Type: text/plain; charset=utf-8

[{"caveats":[{"cid":"object 7zCHWLjyMohzSrKUHRg2wLMb4hvPkV7mdEeDbweAhJZj"}],"location":"","identifier":"76d828f7ae2e3a079c906994304144603cdb6a96d60ef112","signature":"30f1c4c87589e090150912a5b1c13c319c9a7f01100a9c077a14854ff5d3fc4a"}]
```

## POST /:object
Retrieve an object.

### Parameters
- [Path] Location of object given in prior POST. Note that this can also be
  derived from the "object" caveat in the macaroon.
- [Contents] The JSON-encoded macaroon, which is your authorization token for retrieval.

### Response 200 OK
- [Header] Content-Type: _Same content type specified when object was created._
- [Contents] _Object contents._

### Example
```
$ curl -X POST --data @/dev/stdin http://localhost:20080/7zCHWLjyMohzSrKUHRg2wLMb4hvPkV7mdEeDbweAhJZj <<EOF
> [{"caveats":[{"cid":"object 7zCHWLjyMohzSrKUHRg2wLMb4hvPkV7mdEeDbweAhJZj"}],"location":"","identifier":"76d828f7ae2e3a079c906994304144603cdb6a96d60ef112","signature":"30f1c4c87589e090150912a5b1c13c319c9a7f01100a9c077a14854ff5d3fc4a"}]
> EOF
good things$ 
```

## DELETE /:object
Delete an object.

### Parameters
- [Path] Location of object given in prior POST. Note that this can also be
  derived from the "object" caveat in the macaroon.
- [Contents] The JSON-encoded macaroon, which is your authorization token for deleting the object.

### Response 204 No Content

### Example
```
$ curl -i -X DELETE --data @/dev/stdin http://localhost:20080/7zCHWLjyMohzSrKUHRg2wLMb4hvPkV7mdEeDbweAhJZj <<EOF
[{"caveats":[{"cid":"object 7zCHWLjyMohzSrKUHRg2wLMb4hvPkV7mdEeDbweAhJZj"}],"location":"","identifier":"76d828f7ae2e3a079c906994304144603cdb6a96d60ef11
2","signature":"30f1c4c87589e090150912a5b1c13c319c9a7f01100a9c077a14854ff5d3fc4a"}]
EOF
HTTP/1.1 204 No Content
Date: Sat, 19 Sep 2015 04:44:36 GMT
```

# Build

I recommend using a separate GOPATH for every project, to avoid overlapping
dependency conflicts.

Get the source & deps with `go get github.com/cmars/oostore`.

In `$GOPATH/src/github.com/cmars/oostore`, run tests with `go test`.

Install the `oostore` binary into `$GOPATH/bin` with `go get github.com/cmars/oostore/cmd/oostore`.

# License

Copyright 2015 Casey Marshall.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
