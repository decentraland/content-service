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
 -F 'metadata={"value":"/ipfs/iwjfoaoiaeefwe", "pubkey":"0x959e104e1a4db6317fa58f8295f586e1a978c297"}' \
 -F 'zb2rhkshrs1L7X2LWfdFQuEZv6w9HDpWUgGagLGbVhd8NwbeJ=@demo/scene.json' \
 -F 'assets/zb2rhhvg9yGb4fy8qF5K26Pvbx4H4xyo9SDEv9ag4286NzQX7=@demo/assets/text.txt'

curl 'http://localhost:8000/contents/zb2rhhvg9yGb4fy8qF5K26Pvbx4H4xyo9SDEv9ag4286NzQX7'

curl 'http://localhost:8000/validate?x=-40&y=43'

curl 'http://localhost:8000/validate?x=-40&y=4'
