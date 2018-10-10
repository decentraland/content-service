#!/bin/bash

mkdir -p demo/assets

echo 'something' > demo/assets/test.txt

echo '[
  {
    "name": "Compile systems",
    "kind": "Webpack",
    "file": "./scene.tsx",
    "target": "webworker"
  }
]' > demo/build.json

echo '{
  "name": "dcl-project",
  "version": "1.0.0",
  "description": "My new Decentraland project",
  "scripts": {
    "start": "dcl start",
    "build": "decentraland-compiler build.json",
    "watch": "decentraland-compiler build.json --watch"
  },
  "author": "",
  "license": "MIT",
  "devDependencies": {
    "decentraland-api": "latest"
  }
}' > demo/package.json

echo -n '{
  "display": {
    "title": "suspicious_liskov"
  },
  "owner": "",
  "contact": {
    "name": "",
    "email": ""
  },
  "scene": {
    "parcels": [
      "0,0"
    ],
    "base": "0,0"
  },
  "communications": {
    "type": "webrtc",
    "signalling": "https://rendezvous.decentraland.org"
  },
  "policy": {
    "fly": true,
    "voiceEnabled": true,
    "blacklist": [],
    "teleportPosition": "0,0,0"
  },
  "main": "scene.js"
}' > demo/scene.json

echo 'import * as DCL from '\'decentraland-api\''

export default class SampleScene extends DCL.ScriptableScene {
  async render() {
    return (
      <scene>
        <box position={{ x: 5, y: 0.5, z: 5 }} rotation={{ x: 0, y: 45, z: 0 }} color="#4CC3D9" />
        <sphere position={{ x: 6, y: 1.25, z: 4 }} color="#EF2D5E" />
        <cylinder position={{ x: 7, y: 0.75, z: 3 }} radius={0.5} scale={{ x: 0, y: 1.5, z: 0 }} color="#FFC65D" />
        <plane position={{ x: 5, y: 0, z: 6 }} rotation={{ x: -90, y: 0, z: 0 }} scale={4} color="#7BC8A4" />
      </scene>
    )
  }
}' > demo/scene.tsx

echo '{
  "compilerOptions": {
    "module": "esnext",
    "target": "es2017",
    "emitDecoratorMetadata": true,
    "experimentalDecorators": true,
    "sourceMap": true,
    "moduleResolution": "node",
    "declaration": false,
    "strictFunctionTypes": true,
    "forceConsistentCasingInFileNames": true,
    "noUnusedLocals": true,
    "alwaysStrict": true,
    "allowSyntheticDefaultImports": false,
    "allowUnreachableCode": false,
    "allowUnusedLabels": false,
    "newLine": "LF",
    "stripInternal": true,
    "baseUrl": ".",
    "strict": true,
    "jsx": "react",
    "jsxFactory": "DCL.createElement",
    "removeComments": true,
    "outDir": ".",
    "pretty": true,
    "lib": ["es2017", "dom"]
  },
  "exclude": ["node_modules"]
}' > demo/tsconfig.json


curl 'http://localhost:8000/mappings' \
 -F 'metadata={"value": "QmSVHEzaVUVhv8aqXFjssjra6GqgzuUxHsvHEQbiqrJ9pJ","signature": "0xaef3a671b9620d2a03ee05385fafe4ef2ca0e43f0ae880535d99b5b892c9b3d75839c5e28b67dc1e8892905540ec48434b602949779e94e5cbbfe4ff037656fd1c","pubKey": "0xa08a656ac52c0b32902a76e122d2973b022caa0e","validityType": 0,"validity": "2018-12-12T14:49:14.074000000Z","sequence": 2}' \
 -F 'QmSVHEzaVUVhv8aqXFjssjra6GqgzuUxHsvHEQbiqrJ9pJ=[{"cid": "QmaiT7TzzKVjgJ6PJnovQn9DYrFcFyLnFaBseMdyLHCtX8","name": "assets/"},{"cid": "QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672","name": "assets/test.txt"},{"cid": "QmbGdhmRstTdbNBKxqVbGpjiPxy2A5nqrDLuk9KFmQtwox","name": "build.json"},{"cid": "QmTBetsUR4WC1fUB3oM7sDCBQZiHXrsp4LXarqTnHFZ9on","name": "package.json"},{"cid": "QmYTMBFq77WdSvsceAwVfgitRJhzSzGvKSMt5b61LkzwVt","name": "scene.json"},{"cid": "QmSXv3Qgr8pjoYNXZqMhE5Lo9f8FXpYF5cN7vndXsYqJou","name": "scene.tsx"},{"cid": "Qmdv1drP1dkNFKjX6YqL91Go4mY141ZSFQy311qidk9HJc","name": "tsconfig.json"}]' \
 -F 'QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672=@demo/assets/test.txt' \
 -F 'QmbGdhmRstTdbNBKxqVbGpjiPxy2A5nqrDLuk9KFmQtwox=@demo/build.json' \
 -F 'QmTBetsUR4WC1fUB3oM7sDCBQZiHXrsp4LXarqTnHFZ9on=@demo/package.json' \
 -F 'QmYTMBFq77WdSvsceAwVfgitRJhzSzGvKSMt5b61LkzwVt=@demo/scene.json' \
 -F 'QmSXv3Qgr8pjoYNXZqMhE5Lo9f8FXpYF5cN7vndXsYqJou=@demo/scene.tsx' \
 -F 'Qmdv1drP1dkNFKjX6YqL91Go4mY141ZSFQy311qidk9HJc=@demo/tsconfig.json'


curl 'http://localhost:8000/contents/QmTBetsUR4WC1fUB3oM7sDCBQZiHXrsp4LXarqTnHFZ9on'

curl 'http://localhost:8000/validate?x=-0&y=0'

curl 'http://localhost:8000/validate?x=-40&y=4'

curl 'http://localhost:8000/mappings?nw=-1,1&se=1,-1'
