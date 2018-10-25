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
      "54,-136"
    ],
    "base": "54,-136"
  },
  "communications": {
    "type": "webrtc",
    "signalling": "https://rendezvous.decentraland.today"
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
