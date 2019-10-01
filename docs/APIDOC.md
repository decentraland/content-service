# API Documentation

When walking around the world, the client needs to know what scene to load in each parcel. For this reason, there's an endpoint, `/scene`, that provides
the scene that belongs to each parcel. And later there's an endpoint, `/parcel_info`, that is used for retrieving the scene information. There are also some bandwidth consideration when selecting these abstractions. In this sense, the `/scene` endpoint is expected to be hit multiple times by the client, but the `/parcel_info` should be used with more cautelously.

For uploading content, all the scene must be posted into `/mappings` after calculating its CID and signing it.

### POST /mappings

Updates the content for a scene that belongs to a set of parcels. Requires calculating the IPFS CID

Recieves a request with a `Content-Type:multipart/form-data` query parameter, and with the following parts:

- Metadata: is named `metadata` and has a JSON:

```
{
  "value": <root CID>,
  "signature": <signed root CID>, //hex value with 0x prefix
  "pubKey": <eth address>, //with 0x prefix
  "validityType": <int>, //???
  "validity": <timestamp>, //format: "2018-12-12T14:49:14.074000000Z"
  "sequence": <int> //???
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

### GET /validate

This endpoint fetches the metadata from a parcel. It expects the following query paramaters:

- `x=-13`
- `y=16"`

It returns a JSON as follows:

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
You can use this request's response to validate that the contents of the scene haven't been changed since the parcel owner or update operator signed it. You can also verify that the root CID corresponds to the contents of the folder by downloading each of the files (using the `/contents` endpoint) and generating a new CID for them that matches the root CID.

### GET /contents/{CID}

This endpoint gets a file by its `CID`.


### GET /scenes

It receives two x's values and two y's values that will limit a rectangle. It will get all the root cid of the parcels within that rectangle. If for a scene there are parcels outside the queried rectangle, it will return those parcels as well

* Example

```$ curl -H "Content-type: application/json" "https://content.decentraland.zone/scenes?x1=7&y1=9&x2=9&y2=9"```

Response:
```
[
  {
    "parcel_id":"6,9",
    "scene_cid":"QmQpy26Rt758mozFpndPNE752QyyhSuY6YJ1xmZqJJtNv5"
  },
  {
    "parcel_id":"6,8",
    "scene_cid":"QmQpy26Rt758mozFpndPNE752QyyhSuY6YJ1xmZqJJtNv5"
  },
  {
    "parcel_id":"7,9",
    "scene_cid":"QmQpy26Rt758mozFpndPNE752QyyhSuY6YJ1xmZqJJtNv5"
  },
  {
    "parcel_id":"8,9",
    "scene_cid":"QmVND7pVw9KrXqqvAZkavpFA7Pe5xiWSbXCufMnjoeRUwu"
  },
  {
    "parcel_id":"9,9",
    "scene_cid":"QmT7WfCMCqHcGPM5rh7hnhXr791sAMVQgWqubvyc4BeuBB"
  },
  {
    "parcel_id":"7,8",
    "scene_cid":"QmQpy26Rt758mozFpndPNE752QyyhSuY6YJ1xmZqJJtNv5"
  }
]
```
### GET /parcel_info

With the root cid (hash of the previous request) we can query the `/parcel_info` endpoint

* Example

```$ curl -H "Content-type: application/json" "https://content.decentraland.zone/parcel_info?cids=QmVND7pVw9KrXqqvAZkavpFA7Pe5xiWSbXCufMnjoeRUwu"```

Response

```
[{"scene_cid":"QmVND7pVw9KrXqqvAZkavpFA7Pe5xiWSbXCufMnjoeRUwu","content":{"parcel_id":"8,9","contents":[{"file":"models/Plant_05/Plant_05.glb","hash":"QmVwkiRQNQA2wZ68693jH29eCMYwoUNmSZsNh8LJpBJNG6"},{"file":"models/RockLarge_02/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockMediumMoss_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/ConstructionLadder_01/ConstructionLadder_01.glb","hash":"QmUgLMT1nLYa3WMJW8Wis64vt2DstJFgDZJD5cKhag84qZ"},{"file":"scene.json","hash":"QmWcZJ4xXabrq4TzoRP2nYvbPZfMX8D8Zg7d8AP1ysveoS"},{"file":"models/HTC_Portal/TX_EXodus.png.png","hash":"QmR2DNkLrsHevQiwS5bQ9rsjFc9XWidCQRsNU12inFCo15"},{"file":"models/Flower_03/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Mushroom_02/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockMedium_03/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Mushroom_01/Mushroom_01.glb","hash":"QmPFQV4W7ine8GYEGgx7TrcgQ67E9buf23YnChDQSn6cqM"},{"file":"models/Plant_05/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/TreePine_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockMedium_01/RockMedium_01.glb","hash":"QmZ1VuhSrB73QPD85bFf5Typ81fPBS5amQ7XbR5uhW5TFb"},{"file":"models/RockMediumMoss_01/RockMediumMoss_01.glb","hash":"QmQa6TnJdkULMt4N1hv22uamN7QnoejLsUQSAjszYAdbcQ"},{"file":"models/TreePine_01/TreePine_01.glb","hash":"QmP1eadGzG9kkmUQhjXLnn9oXQz934we5zhUvnwQJHPEEV"},{"file":"models/RockMedium_02/RockMedium_02.glb","hash":"QmaB4WeRc1nBnN8VGejuZXt9bMHdmDGnQK6U1pAfZqRkGb"},{"file":"models/Grass_04/Grass_04.glb","hash":"QmYkGZkPMewjqrdrCYiZYtpjyVSh1aj5QdJjCyZRL2WV8q"},{"file":"models/ConstructionLadder_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Flower_04/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Flower_03/Flower_03.glb","hash":"QmU7rfY5cSf7qmbik8iSFWHfdfgHnhkrMTnGSPaBgNuvpY"},{"file":"models/Mushroom_02/Mushroom_02.glb","hash":"QmcLrsMQNcsvrakQJrjmd1VR6rpHqQ5rRnjr823DfUPnTZ"},{"file":"builder.json","hash":"QmXm1YtjvE9PFDvzzivzsdkmGHH34vi4JsQN9t39qUv84k"},{"file":"models/Bridge_04/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Pond_02/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockMedium_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/BushPatch_01/BushPatch_01.glb","hash":"QmQyj3nVn49BYDjiqgC1C5PKBQLmfCo79wwEo8adwbhZ7Y"},{"file":"models/RockSmall_03/RockSmall_03.glb","hash":"QmeSsZpUZxVbLNyVpiwVrKoaNmGiSVehA78fxjoSoXPDc8"},{"file":"models/Pond_02/Pond_02.glb","hash":"QmXMqSY9Q5zECDXjbkPG2t3mTfRoq2RFge5C8zJPdZGiaR"},{"file":"models/Grass_04/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Flower_04/Flower_04.glb","hash":"QmecsnERbYDiKMHJfeD1Y6CWXBkSyUo4iYKN2iq5ZSHiVP"},{"file":"models/RockMedium_03/RockMedium_03.glb","hash":"QmPV4WN5piBCe4yXHHMQhAwy6NxtyyFhGrfaDkcjMibhiU"},{"file":"models/FloorBaseGrass_01/FloorBaseGrass_01.glb","hash":"QmSyvWnb5nKCaGHw9oHLSkwywvS5NYpj6vgb8L121kWveS"},{"file":"models/RockMedium_02/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockSmallMoss_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockSmall_03/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/FloorBaseGrass_01/Floor_Grass01.png.png","hash":"QmT1WfQPMBVhgwyxV5SfcfWivZ6hqMCT74nxdKXwyZBiXb"},{"file":"models/RockLarge_02/RockLarge_02.glb","hash":"QmWc1bwcQRaMc3goS3ZjAvtnZjsVMbMxWhjNsnUipANY56"},{"file":"models/HTC_Portal/HTC_Portal.glb","hash":"QmcWLpSSCJzvBBraNJ9EGmSrNY9WWKjNq4HuKGkw7d1Dtn"},{"file":"models/BushPatch_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Mushroom_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockSmallMoss_01/RockSmallMoss_01.glb","hash":"QmTnAtKqUJCqfRhNVPL3DzZEe9npt7wvWmoV4gb4Lto52s"},{"file":"bin/game.js","hash":"QmYDgg31AxMXgGkYFFxQaJzRB2EiP8DUtRKdYhdvLqhT1E"},{"file":"models/Bridge_04/Bridge_04.glb","hash":"QmYJSvT3nyzdDASWr9YkmvCGzmqj66mDh6qVfTcv5jYxcp"}],"root_cid":"QmVND7pVw9KrXqqvAZkavpFA7Pe5xiWSbXCufMnjoeRUwu","publisher":"0xb79248c11f1b531f4dcecba0ecaebdd55e51ca6c"}}]
```

The content of the query appears as `{"file": <filename>. "hash": <hash>}`. It is as is used to be in the `/mappings` endpoint

Multiple cids can be queried at the same time with comma separated arguments, as in:

```$ curl -H "Content-type: application/json" "https://content.decentraland.zone/parcel_info?cids=QmVND7pVw9KrXqqvAZkavpFA7Pe5xiWSbXCufMnjoeRUwu,QmQpy26Rt758mozFpndPNE752QyyhSuY6YJ1xmZqJJtNv5"```


## Examples

In the following examples we use the data generated by the `demo.sh` script and a local server.

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
$> curl deployment
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