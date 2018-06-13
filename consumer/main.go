package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	cli "gopkg.in/urfave/cli.v1"
)

type Consumer struct {
	Options   *MQTT.ClientOptions
	Topic     string
	Qos       uint8
	Client    MQTT.Client
	reconnect chan struct{}
	shutdown  chan struct{}
	done      sync.WaitGroup
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

var consumer *Consumer

func (consumer *Consumer) close(seconds uint) {
	if consumer.Client != nil && consumer.Client.IsConnected() {
		consumer.Client.Disconnect(seconds * 1000)
	}
	consumer.Client = nil
}

func (consumer *Consumer) prepareSubscription() error {
	consumer.close(3)
	consumer.Client = MQTT.NewClient(consumer.Options)
	if consumer.reconnect == nil {
		consumer.reconnect = make(chan struct{})
	}
	if consumer.shutdown == nil {
		consumer.shutdown = make(chan struct{})
	}
	if token := consumer.Client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Printf("Start to subscribe to topic %v\n", consumer.Topic)
	go func() {
		consumer.done.Add(1)
		defer func() {
			consumer.done.Done()
		}()
		for {
			token := consumer.Client.Subscribe(consumer.Topic, consumer.Qos, handleMessage)
			if token.WaitTimeout(3 * time.Second) {
				select {
				case <-consumer.shutdown:
					log.Println("Received shutdown instruction, consumer go routine exit.")
					return
				default:
				}
				if token.Error() != nil {
					select {
					case consumer.reconnect <- struct{}{}:
						log.Printf("Encounter error %v, notified to reconnect.\n", token.Error())
						return
					default:
					}
				}
			}
		}
	}()

	go func(consumer *Consumer) {
		consumer.done.Add(1)
		defer func() {
			consumer.done.Done()
		}()
		select {
		case <-consumer.reconnect:
			consumer.close(3)
		reconnect_loop:
			for {
				log.Println("re-connecting ...")
				err := consumer.prepareSubscription()
				if err != nil {
					select {
					case <-time.After(5 * time.Second):
						continue reconnect_loop
					case <-consumer.shutdown:
						log.Printf("Received shutdown instruction, re-connect go routine exit.")
						return
					}
				}
				break
			}
			break
		case <-consumer.shutdown:
			log.Printf("Received shutdown instruction, re-connect go routine exit.")
			break
		}

	}(consumer)
	return nil
}

func handleMessage(client MQTT.Client, msg MQTT.Message) {
	log.Printf("Received message %v from topic %v\n", string(msg.Payload()), msg.Topic())
}

func serveListener(ctx *cli.Context) error {
	if !ctx.GlobalIsSet(MqttUrlFlag.Name) || !ctx.GlobalIsSet(TopicFlag.Name) {
		cli.ShowAppHelpAndExit(ctx, -1)
	}
	consumer = new(Consumer)
	consumer.Options = new(MQTT.ClientOptions)
	consumer.Options.AddBroker(ctx.GlobalString(MqttUrlFlag.Name))
	if ctx.GlobalIsSet(ClientIdFlag.Name) {
		consumer.Options.SetClientID(ctx.GlobalString(ClientIdFlag.Name))
	} else {
		consumer.Options.SetClientID("DemoConsumer")
	}
	if ctx.GlobalIsSet(UserFlag.Name) {
		consumer.Options.SetUsername(ctx.GlobalString(UserFlag.Name))
	}
	if ctx.GlobalIsSet(PasswordFlag.Name) {
		consumer.Options.SetPassword(ctx.GlobalString(PasswordFlag.Name))
	}
	if ctx.GlobalIsSet(CleanSessionFlag.Name) {
		consumer.Options.SetCleanSession(ctx.GlobalBool(CleanSessionFlag.Name))
	}
	consumer.Options.SetAutoReconnect(true)
	consumer.Topic = ctx.GlobalString(TopicFlag.Name)

	consumer.Qos = 0
	if ctx.GlobalIsSet(QosFlag.Name) {
		consumer.Qos = uint8(ctx.GlobalUint(QosFlag.Name))
	}
	consumer.Options.SetPingTimeout(1 * time.Second)
	consumer.Options.SetKeepAlive(2 * time.Second)

	err := consumer.prepareSubscription()

	if err != nil {
		return err
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "sample MQTT consumer application."
	app.Flags = []cli.Flag{MqttUrlFlag, UserFlag, PasswordFlag, ClientIdFlag, CleanSessionFlag, QosFlag, TopicFlag}
	app.Action = serveListener
	err := app.Run(os.Args)
	if err != nil {
		if consumer != nil {
			consumer.close(5)
		}
		log.Fatalf("Connection failed %v\n", err)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	defer signal.Stop(sigchan)
	log.Printf("\nGot interrupt, shutting down...singal=%v\n", <-sigchan)
	if consumer != nil {
		if consumer.shutdown != nil {
			close(consumer.shutdown)
		}
		consumer.done.Wait()
		consumer.close(5)
	}
	log.Println("MQTT Consumer gracefully stopped")
}
