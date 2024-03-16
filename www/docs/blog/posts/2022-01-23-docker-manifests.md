---
date: 2022-01-23
slug: docker-manifests
categories:
  - tutorials
authors:
  - dirien
---

# GoReleaser and Docker Manifests

Let's see how to create Docker manifests with GoReleaser.

<!-- more -->

### Question:

Did you know, that you can create Docker Manifest Layer with GoReleaser?

### No?

In this article we will see what Docker Manifests are and how can use them in
GoReleaser to delivery multi-arch builds under one single tag.

But let us start, with the idea behind Docker Image Manifest.

## What are Docker Image Manifests?

![Image Manifests](https://github.com/goreleaser/goreleaser/assets/245435/380b4907-8d7f-4704-852d-8142c1212e86)

<!-- _[Source](https://ownyourbits.com/2019/05/13/building-docker-containers-in-2019/)_ doesn't exist anymole -->

A Docker manifests describe all the layers inside an image.
And with the help of the manifest we can exact compare two images, independent
from their actual human-readable tag.

Manifests are expressed in JSON and contain all the information about the
different image layers and the architectures.

Docker uses then the manifests to work out if an image is compatible with the
current device architecture.
And then uses this particular informations to determine on how to start new
containers.

Currently the
[manifest schema](https://docs.docker.com/registry/spec/manifest-v2-2)
is at version 2.

A manifest file will declare its schema version and then a list of manifest
entries available for the image.
The entries will then point to a different variation of the image, such as
**amd64** and **arm64**.

You can easily view the image manifest using the docker manifest inspect
command.

This works fine with local images or images stored in a remote registry.

```bash
docker manifest inspect <image>:<version>
```

![Example output of the docker manifest inspect command](https://github.com/goreleaser/goreleaser/assets/245435/90b1f47f-8c3f-41dd-962a-7990e14771a9)
_Example output of the docker manifest inspect command_

## Multi-Arch Builds and Manifests

For a long time Docker did not support multiple image architectures.
You could only run images with the same architecture as they where build for.
With the rise of ARM-based machines this was really a limiting factor.

But with manifests, developers can now support multiple architectures under one
single image tag.
The Docker client itself picks the underlying image version for its particular
platform.
Great and simple!

**Keep in mind**: There should be no changes, other the target architecture or
operating system in the images.
Do not deliver images with completely different functionality under the same
tag.

You can read more about docker manifests here:
[https://docs.docker.com/engine/reference/commandline/manifest/](https://docs.docker.com/engine/reference/commandline/manifest/)

> On side note: docker manifest is still an experimental feature and needs to be
> activated in your Docker client.

## GoReleaser

Now that you know what Docker manifest are, we can give GoReleaser the task of
the heavy lifting and let it create the Docker manifest as part of our release
process.

All you need to do is to add the `docker_manifests` to your `.goreleaser.yaml`.
The most important part is to map `name_template` to the `image_templates` you
created in the `dockers` step.

![Example snippet of a gorelaser.yaml](https://github.com/goreleaser/goreleaser/assets/245435/94f6f3fc-98e0-4d9f-96c0-65851ee07e2f)
_Example snippet of a `.gorelaser.yaml`_

There are some additional flags you can set, e.g. if you have a self-hosted
Docker registry with self-signed certificates, you can pass the insecure flag.

Check out the official GoReleaser documentation
[https://goreleaser.com/customization/docker_manifest/](https://goreleaser.com/customization/docker_manifest/)
for an in-depth overview.

## Summary

With GoReleaser its a breeze to create Docker manifest for your multi arch
builds.
Go try it out!

![](https://cdn-images-1.medium.com/max/2000/0*2blEBypJ9QRvqDsm.jpg)

[https://goreleaser.com/](https://goreleaser.com/customization/docker_manifest/)
