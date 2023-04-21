package restapi

import (
	"encoding/json"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

func scrapOutput(cmd []string) (map[string]interface{}, error) {

	lldpctlOutput, err := exec.Command(cmd[0], cmd[1:]...).Output()
	if err != nil {
		log.Println("lldpctl exited with non-zero status")
		return nil, err
	}

	var lldpctlJson map[string]interface{}
	err = json.Unmarshal(lldpctlOutput, &lldpctlJson)
	if err != nil {
		log.Println("Failed to parse lldpctl output")
		return nil, err
	}

	return lldpctlJson, nil
}

func sourceUpdate() map[string]interface{} {
	cmd := []string{"/usr/bin/lldpctl", "-f", "json"}
	cmdLocal := []string{"/usr/bin/lldpcli", "-f", "json", "show", "chassis"}

	lldpJson, err := scrapOutput(cmd)
	if err != nil {
		log.Println(err)
		return nil
	}

	lldpLocChassisJson, err := scrapOutput(cmdLocal)
	if err != nil {
		log.Println(err)
		return nil
	}
	lldpJson["lldp_loc_chassis"] = lldpLocChassisJson

	log.Println("Data fetched from lldp container \n ========", lldpJson)
	return lldpJson
}

func lldpInfo() map[string]interface{} {
	lldpJson := sourceUpdate()
	lldpMain := make(map[string]interface{})

	lldpMain["lldp"] = make(map[string]interface{})

	lldpMain["lldp"].(map[string]interface{})["state"] = make(map[string]interface{})

	lldpMain["lldp"].(map[string]interface{})["interfaces"] = make(map[string]interface{})

	lldpMain["lldp"].(map[string]interface{})["interfaces"].(map[string]interface{})["interface"] = []map[string]interface{}{}

	lldpMain["lldp"].(map[string]interface{})["state"].(map[string]interface{})["enabled"] = true

	var givenType string
	if lldpJson["lldp_loc_chassis"].(map[string]interface{})["local-chassis"].(map[string]interface{})["chassis"] != nil {
		chassis_list := lldpJson["lldp_loc_chassis"].(map[string]interface{})["local-chassis"].(map[string]interface{})["chassis"].((map[string]interface{}))

		for node_name, chassis_details := range chassis_list {
			lldpMain["lldp"].(map[string]interface{})["state"].(map[string]interface{})["system-name"] = node_name

			if val, ok := chassis_details.(map[string]interface{}); ok {
				lldpMain["lldp"].(map[string]interface{})["state"].(map[string]interface{})["system-description"] = val["desc"]

				chassis_id, _ := val["id"].(map[string]interface{})
				lldpMain["lldp"].(map[string]interface{})["state"].(map[string]interface{})["chassis-id"] = chassis_id["value"]
				givenType = chassis_id["type"].(string)
			}

			// Only need first key in the map
			break
		}
	}

	if givenType == "ifalias" {
		lldpMain["lldp"].(map[string]interface{})["state"].(map[string]interface{})["chassis-id-type"] = "INTERFACE_ALIAS"
	} else if givenType == "local" {
		lldpMain["lldp"].(map[string]interface{})["state"].(map[string]interface{})["chassis-id-type"] = "LOCAL"
	} else if givenType == "mac" {
		lldpMain["lldp"].(map[string]interface{})["state"].(map[string]interface{})["chassis-id-type"] = "MAC_ADDRESS"
	} else if givenType == "ip" {
		lldpMain["lldp"].(map[string]interface{})["state"].(map[string]interface{})["chassis-id-type"] = "NETWORK_ADDRESS"
	} else if givenType == "ifname" {
		lldpMain["lldp"].(map[string]interface{})["state"].(map[string]interface{})["chassis-id-type"] = "INTERFACE_NAME"
	} else if givenType == "portcomp" {
		lldpMain["lldp"].(map[string]interface{})["state"].(map[string]interface{})["chassis-id-type"] = "PORT_COMPONENT"
	}

	lldpData, _ := lldpJson["lldp"].(map[string]interface{})
	interfaces := lldpData["interface"].([]interface{})
	for _, interfaceData_ := range interfaces {
		interfaceMap := interfaceData_.(map[string]interface{})
		var interfaceName string
		for interfaceName, _ = range interfaceMap {
			//fetch interface name only
			break
		}
		interfaceDict := make(map[string]interface{})
		interfaceDict["name"] = interfaceName
		interfaceDict["neighbors"] = make(map[string]interface{})
		interfaceDict["neighbors"].(map[string]interface{})["neighbor"] = []map[string]interface{}{}

		neighborDict := make(map[string]interface{})
		interfaceData := interfaceMap[interfaceName].(map[string]interface{})
		chassisData := interfaceData["chassis"].(map[string]interface{})

		var systemName string
		for systemName, _ = range chassisData {
			// fetch only chassis name
			break
		}
		r_id := interfaceData["rid"].(string)
		neighborDict["id"], _ = strconv.Atoi(r_id)
		neighborDict["capabilities"] = make(map[string]interface{})
		neighborDict["capabilities"].(map[string]interface{})["capability"] = []map[string]interface{}{}
		neighborDict["state"] = make(map[string]interface{})
		neighborDict["state"].(map[string]interface{})["id"] = neighborDict["id"]
		neighborDict["state"].(map[string]interface{})["management-address"] = chassisData[systemName].(map[string]interface{})["mgmt-ip"].([]interface{})[0].(string)
		port_desc, _ := interfaceData["port"].(map[string]interface{})
		neighborDict["state"].(map[string]interface{})["port-description"] = port_desc["descr"].(string)
		neighborDict["state"].(map[string]interface{})["ttl"] = port_desc["ttl"].(string)
		neighborDict["state"].(map[string]interface{})["age"] = interfaceData["age"].(string)
		port_id, _ := port_desc["id"].(map[string]interface{})
		neighborDict["state"].(map[string]interface{})["port-id"] = port_id["value"].(string)
		portType := port_id["type"].(string)
		portTypeToName := map[string]string{
			"ifalias":  "INTERFACE_ALIAS",
			"local":    "LOCAL",
			"mac":      "MAC_ADDRESS",
			"ip":       "NETWORK_ADDRESS",
			"ifname":   "INTERFACE_NAME",
			"portcomp": "PORT_COMPONENT",
		}
		neighborDict["state"].(map[string]interface{})["port-id-type"] = portTypeToName[portType]
		neighborDict["state"].(map[string]interface{})["chassis-id"] = chassisData[systemName].(map[string]interface{})["id"].(map[string]interface{})["value"].(string)
		chassisType := chassisData[systemName].(map[string]interface{})["id"].(map[string]interface{})["type"].(string)
		chassisTypeToName := map[string]string{
			"ifalias":  "INTERFACE_ALIAS",
			"local":    "LOCAL",
			"mac":      "MAC_ADDRESS",
			"ip":       "NETWORK_ADDRESS",
			"ifname":   "INTERFACE_NAME",
			"portcomp": "PORT_COMPONENT",
		}
		neighborDict["state"].(map[string]interface{})["chassis-id-type"] = chassisTypeToName[chassisType]
		neighborDict["state"].(map[string]interface{})["system-description"] = chassisData[systemName].(map[string]interface{})["descr"].(string)
		neighborDict["state"].(map[string]interface{})["system-name"] = systemName

		capabilities := []string{"other", "repeater", "mac-bridge", "wlan-access-point", "router", "telephone", "docsis-cable-device", "station-only"}
		for _, capability := range capabilities {
			name := strings.ToUpper(strings.ReplaceAll(capability, "-", "_"))
			capabilityDict := make(map[string]interface{})
			capabilityDict["name"] = name
			capabilityDict["state"] = make(map[string]interface{})
			capabilityDict["state"].(map[string]interface{})["enabled"] = false
			for _, c := range chassisData[systemName].(map[string]interface{})["capability"].([]interface{}) {
				if c.(map[string]interface{})["type"] == capability {
					capabilityDict["state"].(map[string]interface{})["enabled"] = true
					break
				}
			}
			neighborDict["capabilities"].(map[string]interface{})["capability"] = append(
				neighborDict["capabilities"].(map[string]interface{})["capability"].([]map[string]interface{}),
				capabilityDict)
		}
		interfaceDict["neighbors"].(map[string]interface{})["neighbor"] = append(
			interfaceDict["neighbors"].(map[string]interface{})["neighbor"].([]map[string]interface{}),
			neighborDict)
		lldpMain["lldp"].(map[string]interface{})["interfaces"].(map[string]interface{})["interface"] = append(
			lldpMain["lldp"].(map[string]interface{})["interfaces"].(map[string]interface{})["interface"].([]map[string]interface{}),
			interfaceDict)
	}

	return lldpMain
}
