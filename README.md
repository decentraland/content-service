# Content Service

## Configuration

In the base dir you can find the `config.yml` as follows:

```
server:
  port: '8000'                  # port to use for server
  url: 'http://127.0.0.1:8000'  # server URL (used in the replication script)

s3Storage:
  bucket: ''        # Bucket to use in S3
  acl: ''           # ACL for the bucket and files
  url: ''           # URL for the bucket

localStorage: 'tmp'   # local storage dir

redis:
 address: 'content_service_redis:6379'    # address of the redis server
 password: ''                             # password
 DB: 0
```

**Note**: If you use `s3Storage` you need to set AWS environment variables: `AWS_REGION`, `AWS_ACCESS_KEY`, and `AWS_SECRET_KEY`.

## Running

This service runs inside Fargate.


## Endpoints

Locally: `http://localhost:8000`

In Development: `https://content-service.decentraland.zone`

In Production:  `https://content-service.decentraland.org`


### POST /mappings

This endpoint recieves a request with `Content-Type:multipart/form-data` with the following parts:

- Metadata: is named `metadata` and has a JSON:

```
{
  "value": <root CID>,
  "signature": <signed root CID>,
  "pubKey": <eth address>,
  "validityType": <int>,
  "validity": <timestamp>,
  "sequence": <int>
}
```

- Content: is named `<root CID>` and has a JSON:

```
[
  {"cid": <file CID>, "name": <file path>},
  ...
]
```

- Files: the rest of the parts correspond to the uploaded files, they will be named `<file CID>` and have the `filename` header set to file's name.

### GET /mappings

This endpoint gets all the scenes from an area delimited by a northwest coordinate and a southeast coordinate. It expects the following query paramaters:

- `nw="-13,45"`
- `se="13,-45"`

You will receive a JSON as follows:

```
[
  {
    "parcel_id: "-13,45"
    "contents": {
      <file1>: <file1 CID>,
      <file2>: <file2 CID>,
      ...
    }
  },
  ...
]
```

### GET /validate

This endpoint gets the metadata from a parcel. It expects the following query paramaters:

- `x=-13`
- `y=16"`

You will receive a JSON as follows:

```
{
  "value": <root CID>,
  "signature": <signed root CID>,
  "pubKey": <eth address>,
  "validityType": <int>,
  "validity": <timestamp>,
  "sequence": <int>,
  "root_cid": <root CID>
}
```

### GET /contents/{CID}

This endpoint gets a file by its `CID`.

## Demo

To run the demo script you need to:

1. Have Redis up `$ make ops`
1. Set your AWS environment variables: `AWS_REGION`, `AWS_ACCESS_KEY`, `AWS_SECRET_KEY`
1. Start the demo server `$ make demo`



## Examples

**Note**: Set AWS environment variables: `AWS_REGION`, `AWS_ACCESS_KEY`, and `AWS_SECRET_KEY`.
For the following examples we will use the data generated by the `demo.sh` script and a local server.

To run the service in your desktop yout must:
```
docker build -t content_service_golang:latest .
docker run -d --name content_service_redis -p 6379:6379 --rm redis:4.0.11
docker run --name content_service_golang \
-e AWS_REGION=${AWS_REGION} \
-e AWS_ACCESS_KEY=${AWS_ACCESS_KEY} \
-e AWS_SECRET_KEY=${AWS_SECRET_KEY} -p 8000:8000 \
--rm content_service_golang:latest
```

After, you can run the demo script as:

```
$ ./demo.sh`.
<a href="https://content-service.s3.amazonaws.com/text.txt">Moved Permanently</a>.
```

```bash
$> curl 'http://localhost:8000/mappings' \
  -F 'metadata={"value": "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn","signature": "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b","pubKey": "0xa08a656ac52c0b32902a76e122d2973b022caa0e","validityType": 0,"validity": "2018-12-12T14:49:14.074000000Z","sequence": 2}' \
  -F 'QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn=[{"cid": "QmaiT7TzzKVjgJ6PJnovQn9DYrFcFyLnFaBseMdyLHCtX8","name": "assets/"},{"cid": "QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672","name": "assets/test.txt"},{"cid": "QmbGdhmRstTdbNBKxqVbGpjiPxy2A5nqrDLuk9KFmQtwox","name": "build.json"},{"cid": "QmTBetsUR4WC1fUB3oM7sDCBQZiHXrsp4LXarqTnHFZ9on","name": "package.json"},{"cid": "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4","name": "scene.json"},{"cid": "QmSXv3Qgr8pjoYNXZqMhE5Lo9f8FXpYF5cN7vndXsYqJou","name": "scene.tsx"},{"cid": "Qmdv1drP1dkNFKjX6YqL91Go4mY141ZSFQy311qidk9HJc","name": "tsconfig.json"}]' \
  -F 'QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672=@demo/assets/test.txt' \
  -F 'QmbGdhmRstTdbNBKxqVbGpjiPxy2A5nqrDLuk9KFmQtwox=@demo/build.json' \
  -F 'QmTBetsUR4WC1fUB3oM7sDCBQZiHXrsp4LXarqTnHFZ9on=@demo/package.json' \
  -F 'QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4=@demo/scene.json' \
  -F 'QmSXv3Qgr8pjoYNXZqMhE5Lo9f8FXpYF5cN7vndXsYqJou=@demo/scene.tsx' \
  -F 'Qmdv1drP1dkNFKjX6YqL91Go4mY141ZSFQy311qidk9HJc=@demo/tsconfig.json'
```

```bash
$> curl 'http://localhost:8000/contents/QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672'
something

$> curl 'http://localhost:8000/contents/QmbGdhmRstTdbNBKxqVbGpjiPxy2A5nqrDLuk9KFmQtwox'
[
  {
    "name": "Compile systems",
    "kind": "Webpack",
    "file": "./scene.tsx",
    "target": "webworker"
  }
]
```

```bash
$> curl 'http://localhost:8000/validate?x=-0&y=0'
Not Found

$> curl 'http://localhost:8000/validate?x=54&y=-136'
{
  "pubkey": "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
  "rootcid": "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
  "sequence": "2",
  "signature": "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
  "validity": "2018-12-12T14:49:14.074000000Z",
  "validityType": "0",
  "value": "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn"
}
```

```bash
$> curl 'http://localhost:8000/mappings?nw=53,-135&se=55,-137'
[
  {
    "parcel_id": "54,-136",
    "contents": {
      "build.json": "QmbGdhmRstTdbNBKxqVbGpjiPxy2A5nqrDLuk9KFmQtwox",
      "package.json": "QmTBetsUR4WC1fUB3oM7sDCBQZiHXrsp4LXarqTnHFZ9on",
      "scene.json": "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4",
      "scene.tsx": "QmSXv3Qgr8pjoYNXZqMhE5Lo9f8FXpYF5cN7vndXsYqJou",
      "test.txt": "QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672",
      "tsconfig.json": "Qmdv1drP1dkNFKjX6YqL91Go4mY141ZSFQy311qidk9HJc"
    }
  }
]
```

## Replication

To replicate a `content-service` server run:

```
$ make replicate
```

You will recieve a prompt to input the map coordinates for the NW and SE parcels.

This program connects to the server url provided in `config.yml`. It stores the data files in the dir specified by `localstorage` and populates the Redis instance defined in the `redis` field.
