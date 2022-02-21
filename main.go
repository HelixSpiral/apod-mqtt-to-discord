package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/helixspiral/apod"
)

func main() {
	// Some initial setup with our environment
	discordBotToken := os.Getenv("DISCORD_BOT_TOKEN")
	discordChannelId := os.Getenv("DISCORD_CHANNEL_ID")
	mqttBroker := os.Getenv("MQTT_BROKER")
	mqttClientId := os.Getenv("MQTT_CLIENT_ID")
	mqttTopic := os.Getenv("MQTT_TOPIC")

	// Setup the discord bot
	dg, err := discordgo.New("Bot " + discordBotToken)
	if err != nil {
		panic(err)
	}
	dg.Identify.Intents = discordgo.IntentsNone

	// Connect to Discord
	err = dg.Open()
	if err != nil {
		panic(err)
	}

	// Setup the MQTT client options
	options := mqtt.NewClientOptions().AddBroker(mqttBroker).SetClientID(mqttClientId)
	options.ConnectRetry = true
	options.AutoReconnect = true
	options.OnConnectionLost = func(c mqtt.Client, e error) {
		log.Println("Connection lost")
	}
	options.OnConnect = func(c mqtt.Client) {
		log.Println("Connected")

		t := c.Subscribe(mqttTopic, 2, nil)
		go func() {
			_ = t.Wait()
			if t.Error() != nil {
				log.Printf("Error subscribing: %s\n", t.Error())
			} else {
				log.Println("Subscribed to:", mqttTopic)
			}
		}()
	}
	options.OnReconnecting = func(_ mqtt.Client, co *mqtt.ClientOptions) {
		log.Println("Attempting to reconnect")
	}
	options.DefaultPublishHandler = func(_ mqtt.Client, m mqtt.Message) {
		log.Printf("Received: %s->%s\n", m.Topic(), m.Payload())

		// Unmarshal the received json into a struct
		var apodMsg apod.ApodQueryOutput
		_ = json.Unmarshal(m.Payload(), &apodMsg)

		// Build the message we're going to send to Discord
		messageToSend := buildApodMessage(&apodMsg)

		// Send the message to the specified channel
		msg, err := dg.ChannelMessageSend(discordChannelId, messageToSend)
		if err != nil {
			panic(err)
		}

		// "Crosspost" the message so it goes to all the followers.
		_, err = dg.ChannelMessageCrosspost(msg.ChannelID, msg.ID)
		if err != nil {
			panic(err)
		}

	}

	// Setup the MQTT client with the options we set
	mqttClient := mqtt.NewClient(options)

	// Connect to the MQTT server
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	log.Println("Connected")

	// Block indefinitely until something above errors, or we close out.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM)

	<-sig

	log.Println("Signal caught -> Exit")
	mqttClient.Disconnect(1000)
}

// Build the message for Discord
func buildApodMessage(a *apod.ApodQueryOutput) string {
	var messageToSend string

	// Setup the message to send to Discord
	messageToSend += "```"
	messageToSend += fmt.Sprintf("Title: %s\n\n", a.Title)
	messageToSend += fmt.Sprintf("Date: %s\n\n", a.Date)
	messageToSend += fmt.Sprintf("Explanation: %s\n\n", a.Explanation)

	if a.Copyright != "" {
		messageToSend += fmt.Sprintf("Copyright: %s\n", a.Copyright)
	}

	messageToSend += "```\n"

	switch a.MediaType {
	case "image":
		if a.HdUrl != "" {
			messageToSend += a.HdUrl
		} else {
			messageToSend += a.Url
		}
	case "video":
		messageToSend += a.ThumbnailUrl
		messageToSend += "\n\n"
		messageToSend += a.Url
	}

	return messageToSend
}
