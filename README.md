# Content Service

The content service is used to store and distribute content in decentraland that's manipulated by land owners

Our vision of a decentralized platform can't rely on a single centralized server storing an only copy of the virtual world's content. The Content Service allows anyone to host their own instance of the server that stores the world data. Today, the Decentraland client fetches content from a single instance of the content server that's maintained by us. In the future, we want the client to fetch data from multiple replicas of the content, this will guarantee its availability and fast download speeds, independently of the location.

The content hosted in each server includes all the files that make up Decentraland scenes, including .ts scripts, 3D models, texture images, audio files and videos. Currently, each server stores the entirety of the data of all of Genesis city. In future releases of the content service, you'll be able to host a server that only holds the data for specific parcels, to ensure that your scene is always available without depending on any third party.

Each file stored in the content server is cryptographically signed by the parcel owner's key, and the contents of the file are processed to generate a unique hash code, using the same algorithm that IPFS uses to generate its CIDs.

The content server has endpoints that you can send requests to, to fetch data and to validate the authenticity of the signatures on each file.

## Technical overview

*Note: As the original idea was distributing the content through ipfs, all content is still identified by the corresponding hash or CID (though support for ipfs is on hold).* 

Content is organized by scene and queried by parcel. When querying content, the client must ask for all the scenes that at least partially overlap with a given set of parcels, and then all full scenes need to be downloaded.

For uploading, content must be signed by the owner or update operator of the scene, and CIDs must be calculated for every file in the scene.

The upload process may corrupt scenes, so the client is responsible for checking if a given scene is still valid by checking that all parcels that are used by the scene still belong to the scene. This consideration makes working with this API a little bit tricky.

## Requirements

The following dependencies need to be installed to run a content service server.

- [Go 1.12](https://golang.org/dl/)
- [Docker Compose](https://docs.docker.com/compose/)
## Setup git hooks Environment for development

```
$ make init
```


## Configuration

To configure the service, edit the `config.yml` file, in the base directory.

**Note**: If you use `s3Storage` you need to set AWS environment variables: `AWS_REGION`, `AWS_ACCESS_KEY`, and `AWS_SECRET_KEY`.

## Running

### Local environment setup

This service uses S3. In order to run the full service locally without a dependency on a real s3 bucket you will need to install [Localstack](https://github.com/localstack/localstack)

#### Setup
* Install [Localstack](https://github.com/localstack/localstack)
* Install [AWS CLi](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html)
* Run `$ aws configure` to create some fake credentials.
* Run `$ make dev-env`
* Create local bucket for content
 `$ aws --endpoint-url=http://localhost:4572 s3 mb s3://local-content`
* Create local bucket for mappings
 `$ aws --endpoint-url=http://localhost:4572 s3 mb s3://local-mappings`
* Verify the buckets were created: `http://localhost:8055/#!/infra`
* Set the following env variables:
    ```
    AWS_ACCESS_KEY=123123
    AWS_SECRET_KEY=123123
    AWS_REGION=us-west-1
    ```

### Build Project

```
$ make build
```

### Run Project locally

In order to run the project run

```
$ make run AWS_ACCESS_KEY=123123 AWS_SECRET_KEY=123123 AWS_REGION=us-west-1
```

This will use [Localstack](https://github.com/localstack/localstack) as S3 storage provider.

You can read the rest default configuration from [Documentation](config/config.yml)

In order to overwrite any configuration when you run the service, check the env variables you will need to change in the configuration defined in the [service entry point](cmd/service/main.go) 



## API Documentation

[Documentation](https://github.com/decentraland/content-service/blob/master/docs/APIDOC.md)


## Copyright info
This repository is protected with a standard Apache 2 license. See the terms and conditions in the [LICENSE](https://github.com/decentraland/content-service/blob/master/LICENSE) file.
