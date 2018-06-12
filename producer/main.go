package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	cli "gopkg.in/urfave/cli.v1"
)

type Producer struct {
	Options *MQTT.ClientOptions
	Topic   string
	Qos     uint8
	Client  MQTT.Client
}

var (
	MqttUrlFlag = cli.StringFlag{
		Name:  "url",
		Usage: "MQTT url, for example: tcp://10.222.49.29:1883",
	}
	ClientIdFlag = cli.StringFlag{
		Name:  "id",
		Usage: "MQTT url, for example: tcp://10.222.49.29:1883",
	}
	UserFlag = cli.StringFlag{
		Name:  "user",
		Usage: "Solace client user",
	}
	PasswordFlag = cli.StringFlag{
		Name:  "password",
		Usage: "Solace client password",
	}
	CleanSessionFlag = cli.BoolFlag{
		Name:  "clean",
		Usage: "Set Clean Session (default false)",
	}
	QosFlag = cli.UintFlag{
		Name:  "qos",
		Usage: "The Quality of Service 0,1,2 (default 0)",
	}
	TopicFlag = cli.StringFlag{
		Name:  "topic",
		Usage: "Topic, for example: T/testTopic",
	}
)

var producer *Producer

func (producer *Producer) close(seconds uint) {
	if producer.Client != nil && producer.Client.IsConnected() {
		producer.Client.Disconnect(seconds * 1000)
	}
	producer.Client = nil
}

func (producer *Producer) sendMsg(message string) {
	if token := producer.Client.Publish(producer.Topic, byte(producer.Qos), false, message); token.Wait() && token.Error() != nil {
		log.Printf("Send failed %v\n", token.Error())
	}
}

func readLine(reader io.Reader, f func(string)) {
	fmt.Print("Please input message >>>>>>")
	buf := bufio.NewReader(reader)
	line, err := buf.ReadBytes('\n')
	for err == nil {
		line = bytes.TrimRight(line, "\n")
		if len(line) > 0 {
			if line[len(line)-1] == 13 { //'\r'
				line = bytes.TrimRight(line, "\r")
			}
			f(string(line))
		}
		fmt.Print("\nPlease input message >>>>>>")
		line, err = buf.ReadBytes('\n')
	}

	if len(line) > 0 {
		f(string(line))
	}
}

func (producer *Producer) prepareSender() error {
	if producer.Client != nil {
		producer.Client.Disconnect(3 * 1000)
		producer.Client = nil
	}
	producer.Client = MQTT.NewClient(producer.Options)
	if token := producer.Client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func serveSender(ctx *cli.Context) error {
	if !ctx.GlobalIsSet(MqttUrlFlag.Name) || !ctx.GlobalIsSet(TopicFlag.Name) {
		cli.ShowAppHelpAndExit(ctx, -1)
	}
	producer = new(Producer)
	producer.Options = new(MQTT.ClientOptions)
	producer.Options.AddBroker(ctx.GlobalString(MqttUrlFlag.Name))
	if ctx.GlobalIsSet(ClientIdFlag.Name) {
		producer.Options.SetClientID(ctx.GlobalString(ClientIdFlag.Name))
	} else {
		producer.Options.SetClientID("DemoProducer")
	}
	if ctx.GlobalIsSet(UserFlag.Name) {
		producer.Options.SetUsername(ctx.GlobalString(UserFlag.Name))
	}
	if ctx.GlobalIsSet(PasswordFlag.Name) {
		producer.Options.SetPassword(ctx.GlobalString(PasswordFlag.Name))
	}
	if ctx.GlobalIsSet(CleanSessionFlag.Name) {
		producer.Options.SetCleanSession(ctx.GlobalBool(CleanSessionFlag.Name))
	}
	producer.Options.SetAutoReconnect(true)
	producer.Topic = ctx.GlobalString(TopicFlag.Name)

	producer.Qos = 0
	if ctx.GlobalIsSet(QosFlag.Name) {
		producer.Qos = uint8(ctx.GlobalUint(QosFlag.Name))
	}
	producer.Options.SetPingTimeout(1 * time.Second)
	producer.Options.SetKeepAlive(2 * time.Second)

	err := producer.prepareSender()

	if err != nil {
		return err
	}
	go func() {
		readLine(os.Stdin, func(line string) {
			producer.sendMsg(line)
		})
	}()
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "sample MQTT producer application."
	app.Flags = []cli.Flag{MqttUrlFlag, UserFlag, PasswordFlag, ClientIdFlag, CleanSessionFlag, QosFlag, TopicFlag}
	app.Action = serveSender
	err := app.Run(os.Args)
	if err != nil {
		if producer != nil {
			producer.close(5)
		}
		log.Fatalf("Connection failed %v\n", err)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	defer signal.Stop(sigchan)
	log.Printf("\nGot interrupt, shutting down...singal=%v\n", <-sigchan)
	if producer != nil {
		producer.close(5)
	}
	log.Println("MQTT Producer gracefully stopped")
}
