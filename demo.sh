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

curl 'http://localhost:8000/mappings' -F 'metadata={"value":"/ipfs/iwjfoaoiaeefwe", "pubkey":"0x959e104e1a4db6317fa58f8295f586e1a978c297"}' -F 'koidaoidanf=@demo/scene.json' -F 'assets/iodfiofd=@demo/assets/text.txt'

curl 'http://localhost:8000/contents/text.txt'
