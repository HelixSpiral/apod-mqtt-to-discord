package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/helixspiral/apod"
)

func init() {
	logLevel := &slog.LevelVar{}
	if logLevelEnv := os.Getenv("LOG_LEVEL"); logLevelEnv != "" {
		switch logLevelEnv {
		case "ERROR":
			logLevel.Set(slog.LevelError)
		case "WARN":
			logLevel.Set(slog.LevelWarn)
		case "INFO":
			logLevel.Set(slog.LevelInfo)
		case "DEBUG":
			logLevel.Set(slog.LevelDebug)
		default:
			logLevel.Set(slog.LevelInfo)
		}
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Replace the default logger
	slog.SetDefault(logger)
}

func main() {
	// Some initial setup with our environment
	discordBotToken := os.Getenv("DISCORD_BOT_TOKEN")
	discordChannelId := os.Getenv("DISCORD_CHANNEL_ID")
	mqttBroker := os.Getenv("MQTT_BROKER")
	mqttClientId := os.Getenv("MQTT_CLIENT_ID")
	mqttTopic := os.Getenv("MQTT_TOPIC")

	slog.Info("Starting the MQTT to Discord service", "discord_channel_id", discordChannelId, "mqtt_broker", mqttBroker, "mqtt_client_id", mqttClientId, "mqtt_topic", mqttTopic)

	// Setup the discord bot
	dg, err := discordgo.New("Bot " + discordBotToken)
	if err != nil {
		slog.Error("Error creating Discord session", "error", err)

		os.Exit(1)
	}
	dg.Identify.Intents = discordgo.IntentsNone

	// Connect to Discord
	err = dg.Open()
	if err != nil {
		slog.Error("Error opening connection to Discord", "error", err)

		os.Exit(1)
	}

	// Setup the MQTT client options
	options := mqtt.NewClientOptions().AddBroker(mqttBroker).SetClientID(mqttClientId)
	options.ConnectRetry = true
	options.AutoReconnect = true
	options.OnConnectionLost = func(c mqtt.Client, e error) {
		slog.Warn("Connection lost", "error", e)
	}
	options.OnConnect = func(c mqtt.Client) {
		slog.Info("Connected to MQTT broker")

		t := c.Subscribe(mqttTopic, 2, nil)
		go func() {
			_ = t.Wait()
			if t.Error() != nil {
				slog.Error("Error subscribing", "error", t.Error())
			} else {
				slog.Info("Subscribed to MQTT topic", "mqtt_topic", mqttTopic)
			}
		}()
	}
	options.OnReconnecting = func(_ mqtt.Client, co *mqtt.ClientOptions) {
		slog.Info("Reconnecting to MQTT broker", "mqtt_broker", co.Servers)
	}
	options.DefaultPublishHandler = func(_ mqtt.Client, m mqtt.Message) {
		slog.Debug("Received message", "topic", m.Topic(), "payload", string(m.Payload()))

		// Unmarshal the received json into a struct
		var apodMsg apod.ApodQueryOutput
		_ = json.Unmarshal(m.Payload(), &apodMsg)

		// Build the message we're going to send to Discord
		messageToSend := buildApodMessage(&apodMsg)

		// Send the message to the specified channel
		msg, err := dg.ChannelMessageSend(discordChannelId, messageToSend)
		if err != nil {
			slog.Error("Error sending message to Discord", "error", err)

			return
		}

		// "Crosspost" the message so it goes to all the followers.
		_, err = dg.ChannelMessageCrosspost(msg.ChannelID, msg.ID)
		if err != nil {
			slog.Error("Error crossposting message to Discord", "error", err)

			return
		}

	}

	// Setup the MQTT client with the options we set
	mqttClient := mqtt.NewClient(options)

	// Connect to the MQTT server
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("Error connecting to MQTT broker", "error", token.Error())

		os.Exit(1)
	}
	slog.Info("Connected to MQTT broker", "mqtt_broker", mqttBroker)

	// Block indefinitely until something above errors, or we close out.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM)

	<-sig

	slog.Info("Shutting down")
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
