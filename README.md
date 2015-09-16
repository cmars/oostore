# oostore

An opaque object storage service.

oostore is designed for sharing small amounts of content. In exchange for
posted content, oostore responds with a macaroon that may be used to retrieve
or delete the content.

Caveats may then be added to the macaroon received by the content creator,
before distributing it. Just about any kind of authorization policy imaginable
should be possible to implement, by adding caveats to opaque content macaroons.

# API

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
