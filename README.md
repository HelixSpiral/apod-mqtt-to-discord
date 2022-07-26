APOD Discord Bot
---

This is a Discord bot written for my APOD microservice setup. This bot reads from an MQTT topic that another service publishes to daily when NASA releases the new Astronomy Picture of the Day.

This application is largely environment agnostic and relies on environment variables to determine where it sends data.

The variables used are:

* MQTT_BROKER - The broker for your MQTT server
* MQTT_CLIENT_ID - The client ID to use when connecting to your MQTT server
* MQTT_TOPIC - The MQTT topic to publish to
* DISCORD_BOT_TOKEN - The token of the Discord bot being used
* DISCORD_CHANNEL_ID - The channel to send results to

Build with Docker
---

We use the Docker buildx feature to build multiple architectures: `docker buildx build --platform linux/amd64,linux/arm64 -t ghcr.io/helixspiral/apoddiscordbot:latest .`

If all you need is your arch you can omit the platform specific stuff and just do a normal docker build.

Kubernetes setup
---

We've provided an example config map that can be used to run this. You'll also need to create a k8s secret with the `DISCORD_BOT_TOKEN` in the same namespace.

If the image repo being used is private you'll also need to provide k8s with credentials. You can do this with a secret: `kubectl create secret docker-registry <name> --docker-server=<server> --docker-username=<username> --docker-email=<email> --docker-password=<password> -n <namespace>`

Usage
---

To run this you'll need to have an MQTT broker setup to receive messages from, and have a service running that publishes APOD messages to the MQTT topic.

If you aren't writing your own serivce to do that, you can use the service here: https://github.com/HelixSpiral/apod-to-mqtt