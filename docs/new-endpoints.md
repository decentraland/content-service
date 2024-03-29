# New endpoints

Two new endpoints were added to get the contents.

1. `/scenes`

2. `/parcel_info`

Instead of getting the mappings through the `/mappings` endpoint as before, now we will get the `root_cid` of the scene of the parcels on `/scenes` and later we can get the mappings for that scene in the `parcel_info` endpoint.

### GET /scenes

It receives two x's values and two y's values that will limit a rectangle. It will get all the root cid of the parcels within that rectangle. If for a scene there are parcels outside the queried rectangle, it will return those parcels as well

* Example

```$ curl -H "Content-type: application/json" "https://content.decentraland.zone/scenes?x1=7&y1=9&x2=9&y2=9"```

Response:
```
[
    {"7,9":"QmQpy26Rt758mozFpndPNE752QyyhSuY6YJ1xmZqJJtNv5"},
    {"8,9":"QmVND7pVw9KrXqqvAZkavpFA7Pe5xiWSbXCufMnjoeRUwu"},
    {"9,9":"QmT7WfCMCqHcGPM5rh7hnhXr791sAMVQgWqubvyc4BeuBB"},
    {"7,8":"QmQpy26Rt758mozFpndPNE752QyyhSuY6YJ1xmZqJJtNv5"},
    {"6,9":"QmQpy26Rt758mozFpndPNE752QyyhSuY6YJ1xmZqJJtNv5"},
    {"6,8":"QmQpy26Rt758mozFpndPNE752QyyhSuY6YJ1xmZqJJtNv5"}
]
```
### GET /parcel_info

With the root cid (hash of the previous request) we can query the `/parcel_info` endpoint

* Example

```$ curl -H "Content-type: application/json" "https://content.decentraland.zone/parcel_info?cids=QmVND7pVw9KrXqqvAZkavpFA7Pe5xiWSbXCufMnjoeRUwu"```

Response

```
[{"QmVND7pVw9KrXqqvAZkavpFA7Pe5xiWSbXCufMnjoeRUwu":{"parcel_id":"8,9","contents":[{"file":"models/Mushroom_02/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockMedium_02/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Mushroom_01/Mushroom_01.glb","hash":"QmPFQV4W7ine8GYEGgx7TrcgQ67E9buf23YnChDQSn6cqM"},{"file":"models/Flower_03/Flower_03.glb","hash":"QmU7rfY5cSf7qmbik8iSFWHfdfgHnhkrMTnGSPaBgNuvpY"},{"file":"models/RockLarge_02/RockLarge_02.glb","hash":"QmWc1bwcQRaMc3goS3ZjAvtnZjsVMbMxWhjNsnUipANY56"},{"file":"models/HTC_Portal/HTC_Portal.glb","hash":"QmcWLpSSCJzvBBraNJ9EGmSrNY9WWKjNq4HuKGkw7d1Dtn"},{"file":"models/RockLarge_02/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/BushPatch_01/BushPatch_01.glb","hash":"QmQyj3nVn49BYDjiqgC1C5PKBQLmfCo79wwEo8adwbhZ7Y"},{"file":"models/RockSmall_03/RockSmall_03.glb","hash":"QmeSsZpUZxVbLNyVpiwVrKoaNmGiSVehA78fxjoSoXPDc8"},{"file":"models/FloorBaseGrass_01/FloorBaseGrass_01.glb","hash":"QmSyvWnb5nKCaGHw9oHLSkwywvS5NYpj6vgb8L121kWveS"},{"file":"builder.json","hash":"QmXm1YtjvE9PFDvzzivzsdkmGHH34vi4JsQN9t39qUv84k"},{"file":"models/Flower_03/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockSmallMoss_01/RockSmallMoss_01.glb","hash":"QmTnAtKqUJCqfRhNVPL3DzZEe9npt7wvWmoV4gb4Lto52s"},{"file":"models/Mushroom_02/Mushroom_02.glb","hash":"QmcLrsMQNcsvrakQJrjmd1VR6rpHqQ5rRnjr823DfUPnTZ"},{"file":"models/RockSmall_03/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/TreePine_01/TreePine_01.glb","hash":"QmP1eadGzG9kkmUQhjXLnn9oXQz934we5zhUvnwQJHPEEV"},{"file":"models/RockMedium_01/RockMedium_01.glb","hash":"QmZ1VuhSrB73QPD85bFf5Typ81fPBS5amQ7XbR5uhW5TFb"},{"file":"models/HTC_Portal/TX_EXodus.png.png","hash":"QmR2DNkLrsHevQiwS5bQ9rsjFc9XWidCQRsNU12inFCo15"},{"file":"models/RockMedium_02/RockMedium_02.glb","hash":"QmaB4WeRc1nBnN8VGejuZXt9bMHdmDGnQK6U1pAfZqRkGb"},{"file":"models/RockMedium_03/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockMedium_03/RockMedium_03.glb","hash":"QmPV4WN5piBCe4yXHHMQhAwy6NxtyyFhGrfaDkcjMibhiU"},{"file":"scene.json","hash":"QmWcZJ4xXabrq4TzoRP2nYvbPZfMX8D8Zg7d8AP1ysveoS"},{"file":"models/Flower_04/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Plant_05/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockMediumMoss_01/RockMediumMoss_01.glb","hash":"QmQa6TnJdkULMt4N1hv22uamN7QnoejLsUQSAjszYAdbcQ"},{"file":"models/TreePine_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Flower_04/Flower_04.glb","hash":"QmecsnERbYDiKMHJfeD1Y6CWXBkSyUo4iYKN2iq5ZSHiVP"},{"file":"models/Mushroom_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockMediumMoss_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockSmallMoss_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Grass_04/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/Pond_02/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/RockMedium_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/ConstructionLadder_01/ConstructionLadder_01.glb","hash":"QmUgLMT1nLYa3WMJW8Wis64vt2DstJFgDZJD5cKhag84qZ"},{"file":"models/Pond_02/Pond_02.glb","hash":"QmXMqSY9Q5zECDXjbkPG2t3mTfRoq2RFge5C8zJPdZGiaR"},{"file":"models/Bridge_04/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"models/BushPatch_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"},{"file":"bin/game.js","hash":"QmYDgg31AxMXgGkYFFxQaJzRB2EiP8DUtRKdYhdvLqhT1E"},{"file":"models/Bridge_04/Bridge_04.glb","hash":"QmYJSvT3nyzdDASWr9YkmvCGzmqj66mDh6qVfTcv5jYxcp"},{"file":"models/FloorBaseGrass_01/Floor_Grass01.png.png","hash":"QmT1WfQPMBVhgwyxV5SfcfWivZ6hqMCT74nxdKXwyZBiXb"},{"file":"models/Plant_05/Plant_05.glb","hash":"QmVwkiRQNQA2wZ68693jH29eCMYwoUNmSZsNh8LJpBJNG6"},{"file":"models/Grass_04/Grass_04.glb","hash":"QmYkGZkPMewjqrdrCYiZYtpjyVSh1aj5QdJjCyZRL2WV8q"},{"file":"models/ConstructionLadder_01/file1.png","hash":"QmYACL8SnbXEonXQeRHdWYbfm8vxvaFAWnsLHUaDG4ABp5"}],"root_cid":"QmVND7pVw9KrXqqvAZkavpFA7Pe5xiWSbXCufMnjoeRUwu","publisher":"0xb79248c11f1b531f4dcecba0ecaebdd55e51ca6c"}}]
```

The content of the query appears as `{"file": <filename>. "hash": <hash>}`. It is as is used to be in the `/mappings` endpoint.

Multiple cids can be queried at the same time with comma separated arguments, as in:

```$ curl -H "Content-type: application/json" "https://content.decentraland.zone/parcel_info?cids=QmVND7pVw9KrXqqvAZkavpFA7Pe5xiWSbXCufMnjoeRUwu,QmQpy26Rt758mozFpndPNE752QyyhSuY6YJ1xmZqJJtNv5"```