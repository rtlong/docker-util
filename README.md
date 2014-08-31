# docker-util

This is, currently, a very simple tool to help with some of the tasks I find I often want to do while using Docker, specifically in a development context.

At present, it's just two commands:

- `docker-util cleanup-images` &mdash; removes untagged images; approximately equivalent to:
    ```bash
    $ docker images | grep '<none>' | awk '{ print $1 }' | xargs docker rmi
    ```

- `docker-util cleanup-containers` &mdash; removes all non-running containers based on images without tags. Mostly useful because the `cleanup-images` command will fail to remove images for which there is a container based on the image in question. Usually, I'll run this first, followed by `cleanup-images`.

These two operations are something I find myself doing a lot, while iterating through developing a Dockerfile or other Docker workflow.

## Installing

```bash
go get github.com/rtlong/docker-util
```

Or, if you don't have a Golang development env set up, you could run it in Docker. This is built as a [Trusted Build on the Docker Hub](https://registry.hub.docker.com/u/rtlong/docker-util), so just do this:

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock rtlong/docker-util <command>
```
