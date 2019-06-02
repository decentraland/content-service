# Content Service

The content service is used to store and distribute content in decentraland that's manipulated by land owners

Our vision of a decentralized platform can't rely on a single centralized server storing an only copy of the virtual world's content. The Content Service allows anyone to host their own instance of the server that stores the world data. Today, the Decentraland client fetches content from a single instance of the content server that's maintained by us. In the future, we want the client to fetch data from multiple replicas of the content, this will guarantee its availability and fast download speeds, independently of the location.

The content hosted in each server includes all the files that make up Decentraland scenes, including .ts scripts, 3D models, texture images, audio files and videos. Currently, each server stores the entirety of the data of all of Genesis city. In future releases of the content service, you'll be able to host a server that only holds the data for specific parcels, to ensure that your scene is always available without depending on any third party.

Each file stored in the content server is cryptographically signed by the parcel owner's key, and the contents of the file are processed to generate a unique hash code, using the same algorithm that IPFS uses to generate its CIDs.

The content server has endpoints that you can send requests to, to fetch data and to validate the authenticity of the signatures on each file.

## Technical overview

*Note: As the original idea was distributing the content through ipfs, all content is still identified by the corresponding hash or CID (though support for ipfs is on hold).* 

Content is organized by scene and queried by parcel. When querying content, the client must ask for all the scenes that at least partially overlap with a given set of parcels, and then all full scenes need to be downloaded.

For uploading, content must be signed by the public key of the scene owner or update operator, and CIDs must be calculated for every file in the scene.

The upload process may corrupt scenes, so the client is responsible for checking if a given scene is still valid, by checking that all parcels that are used by the scene still belong to the scene. This consideration makes working with this API a little bit tricky.

## Requirements

The following dependencies need to be installed to run a content service server.

- [Go 1.12](https://golang.org/dl/)
- [Docker Compose](https://docs.docker.com/compose/)
- [Redis](https://redis.io)

## Setup git hooks Environment for development

```
$ make init
```


## Configuration

To configure the service, edit the `config.yml` file, in the base directory.

**Note**: If you use `s3Storage` you need to set AWS environment variables: `AWS_REGION`, `AWS_ACCESS_KEY`, and `AWS_SECRET_KEY`.

## Running

First start Redis:

```
$ make ops
```

Then build the project:

```
$ make build
```

You can instead build and run with a single command:

```
$ make run
```

`make run` starts an instance of the content service server.

Alternatively to docker, you can build and run the server with
```
$ go build .
$ ./content-service
```

## API Documentation

// TODO: link

## Replication

To replicate a `content-service` server run:

```
$ make replicate
```

You will recieve a prompt to input the map coordinates for the NW and SE parcels.

This program connects to the server url provided in `config.yml`. It stores the data files in the dir specified by `localstorage` and populates the Redis instance defined in the `redis` field.

## Copyright info
This repository is protected with a standard Apache 2 license. See the terms and conditions in the [LICENSE](https://github.com/decentraland/content-service/blob/master/LICENSE) file.
