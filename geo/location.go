package geo

import "encoding/json"

type Locator interface {
	Run()
	// GetLocation()
	// PutLocation()
}

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type geocodingResponse struct {
	Results []struct {
		Name         string `json:"name"`
		Country      string `json:"country"`
		CountryCode  string `json:"country_code"`
		City         string `json:"city"`
		Postcode     string `json:"postcode"`
		District     string `json:"district"`
		Suburb       string `json:"suburb"`
		Street       string `json:"street"`
		AddressLine1 string `json:"address_line1"`
		Category     string `json:"category"`
	} `json:"results"`
}

type Geolocation struct {
	Name         string `json:"name"`
	Country      string `json:"country"`
	CountryCode  string `json:"country_code"`
	City         string `json:"city"`
	Postcode     string `json:"postcode"`
	District     string `json:"district"`
	Suburb       string `json:"suburb"`
	Street       string `json:"street"`
	AddressLine1 string `json:"address_line1"`
	Category     string `json:"category"`
}

// Format {"key":..., "value":...} string with key being the ip address
// and the value being the Lat, Lon coordinates
func dbIpAddressInsertString(ip string, c Coordinates) (string, error) {
	// Maps to create the json string
	resmap := make(map[string]string)
	keymap := make(map[string]string)
	valmap := make(map[string]Coordinates)

	keymap["key"] = ip
	valmap["value"] = c

	// Encode the key
	keybytes, err := json.Marshal(keymap)
	if err != nil {
		return "", err
	}

	// Encode the value
	valbytes, err := json.Marshal(valmap)
	if err != nil {
		return "", err
	}

	resmap[string(keybytes)] = string(valbytes)
	// Encode the response
	resbytes, err := json.Marshal(resmap)
	if err != nil {
		return "", err
	}

	return string(resbytes), nil
}

// Format {"key":..., "value":...} string with key being the coordinates
// and the value being the geolocation
func dbGeolocationInsertString(c Coordinates, l Geolocation) (string, error) {
	// Maps to create the json string
	resmap := make(map[string]string)
	keymap := make(map[string]Coordinates)
	valmap := make(map[string]Geolocation)

	keymap["key"] = c
	valmap["value"] = l

	// Encode the key
	keybytes, err := json.Marshal(keymap)
	if err != nil {
		return "", err
	}

	// Encode the value
	valbytes, err := json.Marshal(valmap)
	if err != nil {
		return "", err
	}

	resmap[string(keybytes)] = string(valbytes)
	// Encode the response
	resbytes, err := json.Marshal(resmap)
	if err != nil {
		return "", err
	}

	return string(resbytes), nil
}
