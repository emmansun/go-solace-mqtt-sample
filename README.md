# go-solace-mtqq-sample
go solace mqtt sample use https://github.com/eclipse/paho.mqtt.golang

### Enable MQTT Service for your VPN
https://docs.solace.com/Configuring-and-Managing/Managing-MQTT-Messaging.htm

### Test MQTT using paho cmd 
https://github.com/eclipse/paho.mqtt.golang/blob/master/cmd/sample/main.go

### Command Example:
**In fact, the client id should be mandatory! We supplemented default value if user did not specified it.**

- consumer.exe --url tcp://10.222.49.29:1883 --user devuser --password devpwd --topic T/testTopic
- producer.exe --url tcp://10.222.49.29:1883 --user devuser --password devpwd --topic T/testTopic

### Solace Admin UI info
**Non clean session and Qos = 0**
![Producer & Consumer Clients](https://github.com/emmansun/go-solace-mqtt-sample/blob/master/solace_mqtt_1.png)

![Consumer's subscription](https://github.com/emmansun/go-solace-mqtt-sample/blob/master/solace_mqtt_2.png)

**Subscribe with non-clean session and Qos = 1**

There will be one **durable** and **exclusive** queue created.

![Consumer's subscription](https://github.com/emmansun/go-solace-mqtt-sample/blob/master/solace_mqtt_4.png)

![Consumer's subscription](https://github.com/emmansun/go-solace-mqtt-sample/blob/master/solace_mqtt_3.png)
