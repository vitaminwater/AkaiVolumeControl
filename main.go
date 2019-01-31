package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// bluez_sink.E0_E5_CF_67_F1_E0.a2d

var (
	kvExpr = regexp.MustCompile(`(([A-Z0-9a-z_]+) ?= ?(-?[A-Z0-9_a-z.]+))+`)

	server = flag.String("mqtt_server", "tcp://node.local:1883", "The full url of the MQTT server to connect to ex: tcp://127.0.0.1:1883")

	message_chan chan string
	client       MQTT.Client
)

func processMQTTEvent(mqtt_event map[string]interface{}) {
	evt := mqtt_event["evt"].(string)
	id := mqtt_event["id"].(float64)
	if evt == "pot" && id == 1 {
		value := mqtt_event["v"].(float64)
		volume := (int)(value / 127 * 150)
		cmd := exec.Command("/usr/bin/pactl", "set-sink-volume", "bluez_sink.E0_E5_CF_67_F1_E0.a2dp_sink", fmt.Sprintf("%d%%", volume))
		err := cmd.Run()
		if err != nil {
			log.Fatalf("%v", err)
		}
	}
}

func onMessageReceived(client MQTT.Client, message MQTT.Message) {
	mqtt_event := map[string]interface{}{}
	vars := kvExpr.FindAllStringSubmatch(string(message.Payload()), -1)
	for _, varMatch := range vars {
		varName := varMatch[2]
		varValue := varMatch[3]
		numValue, err := strconv.ParseFloat(varValue, 64)
		if err == nil {
			mqtt_event[varName] = numValue
		} else {
			mqtt_event[varName] = varValue
		}
	}
	processMQTTEvent(mqtt_event)
}

func main() {
	connOpts := MQTT.NewClientOptions().AddBroker(*server).SetClientID("SuperGreenLaptop").SetCleanSession(true)
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	connOpts.OnConnect = func(c MQTT.Client) {
		if token := c.Subscribe("akai", 0, onMessageReceived); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}
	}

	client = MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	select {}
}
