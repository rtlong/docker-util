package main

import (
	"fmt"
	"log"
	"os"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	dockerEngine "github.com/fsouza/go-dockerclient/engine"
	kingpin "gopkg.in/alecthomas/kingpin.v1"
)

var (
	client            *docker.Client
	dockerVersion     *dockerEngine.Env
	debug             bool
	forceRemoveImages bool
	containers        = ContainerList{make(map[string]*docker.APIContainers), make(map[string]*docker.APIContainers)}
	images            = ImageList{make(map[string]*docker.APIImages), make(map[string]*docker.APIImages)}
)

type ImageList struct {
	byID      map[string]*docker.APIImages
	byRepoTag map[string]*docker.APIImages
}

func (l ImageList) Add(image *docker.APIImages) {
	l.byID[shortID(image.ID)] = image

	for _, repoTag := range image.RepoTags {
		if !repoTagIsNone(repoTag) {
			l.byRepoTag[repoTag] = image
		}
	}
}

func (l ImageList) FindByID(id string) (img *docker.APIImages, ok bool) {
	img, ok = l.byID[id]
	return
}

func (l ImageList) FindByRepoTag(repoTag string) (img *docker.APIImages, ok bool) {
	img, ok = l.byRepoTag[repoTag]
	return
}

func (l ImageList) Iterator() chan *docker.APIImages {
	out := make(chan *docker.APIImages)
	go func() {
		for _, img := range l.byID {
			out <- img
		}
		close(out)
	}()
	return out
}

func (l ImageList) Print() {
	for k, v := range l.byID {
		fmt.Printf("%s =>\n    %s\n", k, v)
	}
}

type ContainerList struct {
	byID   map[string]*docker.APIContainers
	byName map[string]*docker.APIContainers
}

func (l ContainerList) FindByID(id string) (c *docker.APIContainers, ok bool) {
	c, ok = l.byID[id]
	return
}

func (l ContainerList) FindByName(name string) (c *docker.APIContainers, ok bool) {
	c, ok = l.byName[name]
	return
}

func (l ContainerList) Add(container *docker.APIContainers) {
	l.byID[shortID(container.ID)] = container

	for _, name := range container.Names {
		l.byName[name] = container
	}
}

func (l ContainerList) Iterator() chan *docker.APIContainers {
	out := make(chan *docker.APIContainers)
	go func() {
		for _, img := range l.byID {
			out <- img
		}
		close(out)
	}()
	return out
}

func (l ContainerList) Print() {
	for k, v := range l.byID {
		fmt.Printf("%s =>\n    %s\n", k, v)
	}
}

func getContainers(cntList *ContainerList) {
	options := docker.ListContainersOptions{All: true}
	cnts, err := client.ListContainers(options)
	if err != nil {
		log.Fatal("client.ListContainers(", options, "): ", err)
	}
	for i := range cnts {
		cntList.Add(&cnts[i])
	}
	if debug {
		cntList.Print()
	}
}

func getImages(imgList *ImageList) {
	imgs, err := client.ListImages(false)
	if err != nil {
		log.Fatal("client.ListImages(false): ", err)
	}
	for i := range imgs {
		imgList.Add(&imgs[i])
	}
	if debug {
		imgList.Print()
	}
}

func createClient() {
	var (
		dockerEndpoint string
		err            error
	)

	if dockerEndpoint = os.Getenv("DOCKER_HOST"); len(dockerEndpoint) == 0 {
		dockerEndpoint = "unix:///var/run/docker.sock"
	}

	if debug {
		log.Println("Connecting to Docker daemon via:", dockerEndpoint)
	}

	client, err = docker.NewClient(dockerEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	dockerVersion, err = client.Version()
	if err != nil {
		log.Fatal("Could not connect to Docker daemon: ", err)
	}
	if debug {
		showDockerVersion()
	}
}

func showDockerVersion() {
	fmt.Println("Docker daemon info:")
	for key, value := range dockerVersion.Map() {
		fmt.Printf("  %-15s : %s\n", key, value)
	}
}

func cleanupImages() {
	getImages(&images)

	for img := range images.Iterator() {
		if len(img.RepoTags) == 1 && repoTagIsNone(img.RepoTags[0]) {
			log.Printf("removing image %s", shortID(img.ID))
			if err := client.RemoveImage(img.ID); err != nil {
				log.Println(err)
			}
		}
	}
}

func cleanupContainers() {
	getContainers(&containers)
	getImages(&images)

	var options = docker.RemoveContainerOptions{RemoveVolumes: true}
	for c := range containers.Iterator() {
		// if the Image doesn't refer to a repoTag, the container should be removed
		if _, ok := images.FindByRepoTag(c.Image); !ok {
			log.Printf("removing container %s based on image %s", shortID(c.ID), c.Image)
			options.ID = c.ID
			if err := client.RemoveContainer(options); err != nil {
				log.Println(err)
			}
		}
	}
}

func logDockerEvents(events <-chan *docker.APIEvents) {
	for {
		e := <-events
		log.Printf("daemon: %s %s %s", e.From, e.Status, e.ID)
	}
}

func repoTagIsNone(tagName string) bool {
	return tagName == "<none>:<none>"
}

func shortID(id string) string {
	return id[0:12]
}

func main() {
	kingpin.Flag("debug", "Show extra info").BoolVar(&debug)
	kingpin.Command("cleanup-images", "Cleanup any untagged images")
	kingpin.Command("cleanup-containers", "Cleanup any containers based on untagged images")
	command := kingpin.Parse() // get selected command or die here

	createClient()

	// Log events from the Daemon
	var events = make(chan *docker.APIEvents)
	client.AddEventListener(events)
	go logDockerEvents(events)

	switch command {
	case "cleanup-images":
		cleanupImages()
	case "cleanup-containers":
		cleanupContainers()
	}

	// FIXME: Wait a short moment for any last Daemon events to come back
	time.Sleep(time.Second * 1)
}
