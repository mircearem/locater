package geo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/mircearem/storer/store"
	"github.com/sirupsen/logrus"
)

type Ip2LocStruct struct {
	Connection struct {
		IP        string `json:"ip"`
		IPVersion string `json:"ip_version"`
	} `json:"connection"`
	Currency struct {
		Code []string `json:"code"`
	} `json:"currency"`
	Location struct {
		Capital   string `json:"capital"`
		City      string `json:"city"`
		Continent struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"continent"`
		Country struct {
			Alpha2        string   `json:"alpha_2"`
			Alpha3        string   `json:"alpha_3"`
			DialingCode   []string `json:"dialing_code"`
			Emoji         string   `json:"emoji"`
			EuMember      bool     `json:"eu_member"`
			Name          string   `json:"name"`
			Subdivision   string   `json:"subdivision"`
			SubdivisionID string   `json:"subdivision_id"`
			ZipCode       string   `json:"zip_code"`
		} `json:"country"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
	Success bool `json:"success"`
	Time    struct {
		Zone string `json:"zone"`
	} `json:"time"`
}

// Locator type
type LanLocator struct {
	Ip     string
	Locch  chan struct{}    // channel that signals that it is time to check for a location change
	Sendch chan Geolocation // channel used to update the location on the server
	db     *store.Client    // database that stores locations
	// channels signaling that the ip has been updated from the api
	newipch chan struct{} // a new ip, not priorly found in the cache or in the db
	mapipch chan struct{} // a new ip that is found in the map
	dbipch  chan struct{} // a new ip that is found in the db
	// Caches of known ips and locations
	mu   sync.RWMutex
	ips  map[string]Coordinates
	locs map[Coordinates]Geolocation
}

func NewLanLocator(locch chan struct{}, sendch chan Geolocation) *LanLocator {
	client := store.NewClient("localhost:7777")
	return &LanLocator{
		db:      client,
		Locch:   locch,
		Sendch:  sendch,
		ips:     make(map[string]Coordinates),
		locs:    make(map[Coordinates]Geolocation),
		newipch: make(chan struct{}),
		mapipch: make(chan struct{}),
	}
}

func (l *LanLocator) Run() {
	go l.handleOldLocation()
	go l.handleNewLocation()

	for {
		<-l.Locch
		err := l.getIpAddress()
		if err != nil {
			continue
		}
		l.mu.RLock()
		// Ip is in the map, it was already registered; retrieve
		// the coordinates and the geolocation from the map
		if _, ok := l.ips[l.Ip]; ok {
			l.mapipch <- struct{}{}
			l.mu.RUnlock()
			continue
		}
		l.mu.RUnlock()
		// Ip is not in the map, go check the database
		l.dbipch <- struct{}{}
	}
}

// Location already in map or database, signal received
// through the <-oldipch
func (l *LanLocator) handleOldLocation() {
	for {
		select {
		case <-l.mapipch:
			// Ip address was found in the map
			// get the relevant information from the map
			l.mu.RLock()
			c := l.ips[l.Ip]
			geo := l.locs[c]
			l.mu.RUnlock()
			l.Sendch <- geo
		case <-l.dbipch:
			// Ip address was not found in the map, go check the database
			latlon, err := l.db.Get("remoteaddr", l.Ip)
			if err != nil || latlon == "" {
				l.newipch <- struct{}{}
				continue
			}
			// Ip address found in the database, use the coordinates to get the geolocation
			c, err := l.db.Get("locations", latlon)
			if err != nil || c == "" {
				l.newipch <- struct{}{}
				continue
			}
			var geo Geolocation
			if err := json.Unmarshal([]byte(c), &geo); err != nil {
				continue
			}
			l.Sendch <- geo
		}
	}
}

// New ip address found, put it into the map and database
// signal received through the <-newipch
func (l *LanLocator) handleNewLocation() {
	for {
		<-l.newipch
		// New registered ip, get coordinates and put it in the map
		// and the database
		c, err := l.getLatLon()
		if err != nil {
			logrus.Println(err)
			continue
		}
		// Geolocate
		geo, err := l.getGeolocation(c)
		if err != nil {
			logrus.Println(err)
			continue
		}
		// Add the new data to the map - use map as cache backup to not query the db so often
		l.mu.Lock()
		l.ips[l.Ip] = c
		l.locs[c] = geo
		l.mu.Unlock()
		// Add the data to the database
		// Format key value pair for ip - coordinates
		str, err := dbIpAddressInsertString(l.Ip, c)
		if err != nil {
			logrus.Println(err)
			continue
		}
		// Insert the ip - coordinates pair in the db
		if err = l.db.Post("remoteaddr", []byte(str)); err != nil {
			logrus.Println(err)
			continue
		}
		// Format key value pair for coordinates - geolocation
		str, err = dbGeolocationInsertString(c, geo)
		if err != nil {
			logrus.Println(err)
			continue
		}
		// Insert the coordinates - geolocation pair in the db
		if err = l.db.Post("geolocation", []byte(str)); err != nil {
			logrus.Println(err)
			continue
		}
		l.Sendch <- geo
	}
}

// Get the geolocation using
func (l *LanLocator) getGeolocation(c Coordinates) (Geolocation, error) {
	API_URI := os.Getenv("GEOCODING_API_URI")
	API_KEY := os.Getenv("GEOCODING_API_KEY")
	url := fmt.Sprintf("%slat=%f&lon=%f&format=json&apiKey=%s", API_URI, c.Lat, c.Lon, API_KEY)
	// Call the api
	// @TODO: add a client other the DefaultClient
	res, err := http.Get(url)
	if err != nil {
		msg := fmt.Sprintf("geolocation response fail: %s", err.Error())
		return Geolocation{}, errors.New(msg)
	}
	defer res.Body.Close()
	// Decode the message
	var loc geocodingResponse
	if err := json.NewDecoder(res.Body).Decode(&loc); err != nil {
		msg := fmt.Sprintf("cannot parse API response: %s", err)
		return Geolocation{}, errors.New(msg)
	}
	geo := loc.Results[0]
	return geo, nil
}

// Get location using ip2loc
func (l *LanLocator) getLatLon() (Coordinates, error) {
	API_URI := os.Getenv("IPLOCATION_API_URI")
	API_KEY := os.Getenv("IPLOCATION_API_KEY")
	url := fmt.Sprintf("%s/%s/%s", API_URI, API_KEY, l.Ip)

	res, err := http.Get(url)
	if err != nil {
		msg := fmt.Sprintf("ip2loc response fail: %s", err.Error())
		return Coordinates{}, errors.New(msg)
	}
	defer res.Body.Close()
	// Decode the message
	latlon := new(Ip2LocStruct)
	if err := json.NewDecoder(res.Body).Decode(latlon); err != nil {
		msg := fmt.Sprintf("cannot parse API response: %s", err)
		return Coordinates{}, errors.New(msg)
	}
	return Coordinates{
		Lat: latlon.Location.Latitude,
		Lon: latlon.Location.Longitude,
	}, nil
}

// Get the IP address using ipify
func (l *LanLocator) getIpAddress() error {
	// Get the IP address of the server
	API_URI := os.Getenv("IPIFY_API_URI")
	res, err := http.Get(API_URI)

	// Check for request errors
	if err != nil {
		return fmt.Errorf("ipify response fail: %s", err.Error())
	}
	defer res.Body.Close()

	// Check error when reading body
	if err != nil {
		return fmt.Errorf("ipify response parsing fail: %s", err.Error())
	}
	respJson := make(map[string]interface{})

	err = json.NewDecoder(res.Body).Decode(&respJson)
	// Check error when unmarshalling json
	if err != nil {
		return fmt.Errorf("json unmarshall error: %s", err.Error())
	}
	ip := respJson["ip"].(string)
	l.Ip = ip

	return nil
}

// Run these functions as go routines in the background
// listening to channels to begin querying the database
// for information
func (l *LanLocator) FindIpAddr()      {}
func (l *LanLocator) FindCoordinates() {}
func (l *LanLocator) PutIpAddr()       {}
func (l *LanLocator) PutCoordinates()  {}
