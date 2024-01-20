package geo

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/mircearem/locater/modem"
	"github.com/mircearem/storer/store"
)

var (
	OPENCELLID_API_URI string
	OPENCELLID_API_KEY string
)

type CellularLocator struct {
	m        *modem.Modem
	db       *store.Client
	c        Coordinates
	Sendch   chan Geolocation
	Locch    chan struct{}
	latlonch chan struct{}
	mu       sync.RWMutex
	locs     map[Coordinates]Geolocation
}

func NewCellLocator(m *modem.Modem, Sendch chan Geolocation, Locch chan struct{}) *CellularLocator {
	client := store.NewClient("localhost:7777")
	return &CellularLocator{
		m:      m,
		db:     client,
		Sendch: Sendch,
		Locch:  Locch,
	}
}

func (l *CellularLocator) Run() {
	go l.geolocate()

	for {
		<-l.Locch
		// Call the OpenCellId API
		err := l.getLatLon()
		if err != nil {
			continue
		}
		// Check if the location is already in the map
		l.mu.RLock()
		if _, ok := l.locs[l.c]; !ok {
			l.latlonch <- struct{}{}
			l.mu.RUnlock()
			continue
		}
		l.mu.RUnlock()
		log.Printf("Coordinates already in map: %+v\n", l.c)
		// Get the location using the Geocoding API
		geo, err := l.getGeolocation(l.c)
		if err != nil {
			continue
		}
		// Send the new geolocation back to the server
		l.Sendch <- geo
	}
}

// Geolocate
func (l *CellularLocator) geolocate() {
	for {
		<-l.latlonch
		// New registered ip, get coordinates and put it in the map
		err := l.getLatLon()
		if err != nil {
			log.Println(err)
			continue
		}
		// Geolocate
		geo, err := l.getGeolocation(l.c)
		if err != nil {
			log.Println(err)
			continue
		}
		// // Add the new data to the map
		// l.mu.Lock()
		// l.locs[l.c] = geo
		// l.mu.Unlock()
		// Format the data to {"key": "value"} maybe use function
		// Add the new location to tha database
		err = l.db.Post("locations", nil)
		if err != nil {
			log.Println(err)
			continue
		}
		l.Sendch <- geo
	}
}

// Get the geolocation using the geocoding api
func (l *CellularLocator) getGeolocation(c Coordinates) (Geolocation, error) {
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

// Get the location coordinates using the OpenCellId API
func (l *CellularLocator) getLatLon() error {
	url := fmt.Sprintf(`%s?key=%s&mcc=%d&mnc=%d&lac=%d&cellid=%d&format=json`, OPENCELLID_API_URI, OPENCELLID_API_KEY, l.m.Network.Mcc, l.m.Network.Mnc, l.m.Network.Lac, l.m.Network.Cid)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Parse the response
	if err := json.NewDecoder(resp.Body).Decode(&l.c); err != nil {
		return err
	}
	return nil
}
