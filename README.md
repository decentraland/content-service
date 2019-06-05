# Content Service

The content service is used to store and distribute content in decentraland that's manipulated by land owners

## Technical overview

*Note: As the original idea was distributing the content through ipfs, all content is still identified by the corresponding hash or CID (though support for ipfs is on hold).* 

Content is organized by scene and queried by parcel. When querying content, the client must ask for all the scenes that intersect a region of parcels, and then all full scenes need to be downloaded.

For uploading, content must be signed and all CIDs calculated

The upload process may corrupt scenes, so the client is responsible for checking if a given scene is still valid by checking that all parels of the scene still belong to the scene. This make working with this API a little bit tricky.

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

[Documentation](https://github.com/decentraland/content-service/blob/master/docs/APIDOC.go)

## Replication

To replicate a `content-service` server run:

```
$ make replicate
```

You will recieve a prompt to input the map coordinates for the NW and SE parcels.

This program connects to the server url provided in `config.yml`. It stores the data files in the dir specified by `localstorage` and populates the Redis instance defined in the `redis` field.

## Copyright info
This repository is protected with a standard Apache 2 license. See the terms and conditions in the [LICENSE](https://github.com/decentraland/content-service/blob/master/LICENSE) file.
