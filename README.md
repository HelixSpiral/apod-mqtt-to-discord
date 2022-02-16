APOD Discord Bot
---

This is a Discord bot written for my APOD microservice setup. This bot reads from an MQTT topic that another service publishes to daily when NASA releases the new Astronomy Picture of the Day.

Usage
---

This application is largely environment agnostic and relies on environment variables to determine where it sends data.

The variables used are:

* MQTT_BROKER - The broker for your MQTT server
* MQTT_CLIENT_ID - The client ID to use when connecting to your MQTT server
* MQTT_TOPIC - The MQTT topic to publish to
* DISCORD_BOT_TOKEN - The token of the Discord bot being used
* DISCORD_CHANNEL_ID - The channel to send results to