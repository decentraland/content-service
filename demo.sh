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
 -F 'metadata={"value":"/ipfs/QmZHXazb2PUGco1nTtxFYNywP5z9Ewqj2m5bGiW2g1G3bJ", "pubkey":"0x959e104e1a4db6317fa58f8295f586e1a978c297", "signature":"0x09266906d0351fe9c17beb4ffe3fb3db30788e84c10dff3db0eb5cec120f8cdd598ab7fb699a6a5e0c694fe1253cc63562bfbe27a089d56ec86d484348534fb01c"}' \
 -F 'content=[{"cid":"QmQRnSrVGQL7p2QiE7ShsQspyc7asfdBY44LkmxM8aKHob", "name":"scene.json"},{"cid":"QmURx54X51QdYjgzme4QfHnEGKej4XCuwSzBxEAwyHwgoH", "name":"assets/text.txt"}' \
 -F 'QmQRnSrVGQL7p2QiE7ShsQspyc7asfdBY44LkmxM8aKHob=@demo/scene.json' \
 -F 'assets/QmURx54X51QdYjgzme4QfHnEGKej4XCuwSzBxEAwyHwgoH=@demo/assets/text.txt'

curl 'http://localhost:8000/contents/QmURx54X51QdYjgzme4QfHnEGKej4XCuwSzBxEAwyHwgoH'

curl 'http://localhost:8000/validate?x=-40&y=43'

curl 'http://localhost:8000/validate?x=-40&y=4'

curl 'http://localhost:8000/mappings?nw=-40,43&se=-39,43'

curl 'http://localhost:8000/mappings' \
 -F 'metadata={"value":"/ipfs/QmZHXazb2PUGco1nTtxFYNywP5z9Ewqj2m5bGiW2g1G3bJ", "pubkey":"0x71679b015d5f0a100D736e4033B220C502bB022b", "signature":"0x19b108c5da56be28da4cc5adc09e152b08f5f34dc562a22b3943cd1ce4dc7d9f6b2514a012764ad37820ae87b18826031659372ffdf30ea8cce3ae075d2fe11c01"}' \
 -F 'content=[{"cid":"QmQRnSrVGQL7p2QiE7ShsQspyc7asfdBY44LkmxM8aKHob", "name":"scene.json"},{"cid":"QmURx54X51QdYjgzme4QfHnEGKej4XCuwSzBxEAwyHwgoH", "name":"assets/text.txt"}' \
 -F 'QmQRnSrVGQL7p2QiE7ShsQspyc7asfdBY44LkmxM8aKHob=@demo/scene.json' \
 -F 'assets/QmURx54X51QdYjgzme4QfHnEGKej4XCuwSzBxEAwyHwgoH=@demo/assets/text.txt'
