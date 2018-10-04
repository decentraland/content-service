#!/bin/bash

mkdir -p demo/assets

echo '{
  "display": {
    "title": "distracted_roentgen"
  },
  "owner": "ADDRESS",
  "scene": {
    "estateId": 219,
    "parcels": ["-39,43", "-40,43"],
    "base": "-34,-130"
  },
  "communications": {
    "type": "webrtc",
    "signalling": "https://rendezvous.decentraland.org"
  },
  "main": "scene.js"
}' > demo/scene.json

echo 'This is a text file' > demo/assets/text.txt


curl 'http://localhost:8000/mappings' \
 -F 'metadata={"value":"/ipfs/QmZHXazb2PUGco1nTtxFYNywP5z9Ewqj2m5bGiW2g1G3bJ", "pubkey":"0x959e104e1a4db6317fa58f8295f586e1a978c297"}' \
 -F 'QmQRnSrVGQL7p2QiE7ShsQspyc7asfdBY44LkmxM8aKHob=@demo/scene.json' \
 -F 'assets/QmURx54X51QdYjgzme4QfHnEGKej4XCuwSzBxEAwyHwgoH=@demo/assets/text.txt'

curl 'http://localhost:8000/contents/QmURx54X51QdYjgzme4QfHnEGKej4XCuwSzBxEAwyHwgoH'

curl 'http://localhost:8000/validate?x=-40&y=43'

curl 'http://localhost:8000/validate?x=-40&y=4'

curl 'http://localhost:8000/mappings?nw=-40,43&se=-39,43'
