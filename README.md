![build](https://github.com/supby/gigbee2mqtt/actions/workflows/go.yml/badge.svg)

# Zigbee to MQTT gateway

Acts like gateway between MQTT and Zigbee devices. Unlike zigbee2mqtt it works on ZCL level using messages format as they defined in ZCL specification.
In order to build some automation you need to know device ZCL information (cluster, endpoint)

## MQTT topics and messages

**Device state reporting**

Device reports on topic `gigbee2mqtt/<device addr>`

Example:
```
gigbee2mqtt/0x00124b00217301e4
{
    "IEEEAddress":5149013072719364,
    "LinkQuality":31,
    "Message":{
        "ClusterID":1280,
        "ClusterName":"ssIasZone",
        "ClusterType":"",
        "ClusterAttributes":{
            "zoneStatus":0
        }
    }
}
```

**Device state get**

In order to retreive device state following message should be sent on topic `gigbee2mqtt/<device addr>/get`:
```
{
  "ClusterID": <zcl cluster id>,
  "Endpoint": <device endpoint>,
  "Attributes": [<list of attributes to retreive>]
}
```

Example:
```
// OnOff get
gigbee2mqtt/0x842e14fffe05b879/get
{
  "ClusterID": 6,
  "Endpoint": 1,
  "Attributes": [0]
}
```

**Device state set**

In order to set device state, following message should be sent on topic `gigbee2mqtt/<device addr>/set`:
```
{
  "ClusterID": <zcl cluster id>,
  "Endpoint": <endpoint to set>,
	"CommandIdentifier": <id of command>,
	"CommandData": <object with command data to send according to ZCL specification>
}
```

Example:
```
// Set level
gigbee2mqtt/0x00124b00217301e4/set
{
  "ClusterID": 8,
  "Endpoint": 6,
	"CommandIdentifier": 4,
	"CommandData": {
        "Level": 108,
	 	"TransitionTime": 1
  }
}
```
**Explore device**

In order to get device description, send empty message on topic `gigbee2mqtt/<device addr>/explore`.
Response will be received on topic `gigbee2mqtt/<device addr>/description` in format:
```
{
  "IEEEAddress": <device address>,
  "LogicalType": <logical type>,
  "ManufacturerCode": <int manufacturer code>,
  "Endpoints": [ // list of enpoints supported by device
    {
      "Endpoint": <endpoint number>,
      "ProfileID": <ZB profile id>,
      "DeviceID": <devie ID>,
      "DeviceVersion": <device version>,
      "InClusterList":[<list of inbound clusters>],
      "OutClusterList":[<list of outbound clusters>]
    }]
}

```
Example:
```
gigbee2mqtt_dev/0x842e14fffe05b879/description
{
  "IEEEAddress":9524573351646181497,
  "LogicalType":1,
  "ManufacturerCode":4098,
  "Endpoints": [
    {
      "Endpoint":1,
      "ProfileID":260,
      "DeviceID":9,
      "DeviceVersion":1,
      "InClusterList":[0,4,5,6],
      "OutClusterList":[25,10]
    }]
}
```

**Get list of joined devices**

Send empty object to `gigbee2mqtt/gateway/get_devices`

**Get gateway config**

Send empty object to `gigbee2mqtt/gateway/get_config`

**Set gateway config**

Send object to `gigbee2mqtt/gateway/set_config`
```
{
    "PermitJoin": <true/false>
}
```

For now only `PermitJoin` can be changed.


## Configuration

Example of configuration:
```
znetworkconfiguration:
  panid: 1819
  extendedpanid: 11960156591840108824
  networkkey:
  - 1
  - 3
  - 5
  - 7
  - 9
  - 15
  - 13
  - 15
  - 0
  - 1
  - 4
  - 6
  - 7
  - 10
  - 14
  - 13
  channel: 18
mqttconfiguration:
  address: 192.168.1.25
  port: 1883
  roottopic: gigbee2mqtt
  username: ""
  password: ""
serialconfiguration:
  portname: /dev/ttyACM0
  baudrate: 115200
permitjoin: true
```
