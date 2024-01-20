package geo

import (
	"context"
	"log"
	"time"

	"github.com/mircearem/locater/modem"
)

// Embed the database into the server
type Server struct {
	Location  Geolocation
	ctx       context.Context
	m         *modem.Modem
	locRecvch chan Geolocation
	locch     chan struct{}
	quitch    chan struct{}
}

func NewServer(ctx context.Context) *Server {
	m, err := modem.NewModem(ctx)
	// No modem is available, geolocation done using ip2loc solution
	if err != nil {
		return &Server{
			m:         nil,
			ctx:       ctx,
			locch:     make(chan struct{}),
			locRecvch: make(chan Geolocation),
			quitch:    make(chan struct{}),
		}
	}
	// A modem is available, geolocation done using OpenCellId
	return &Server{
		m:         m,
		ctx:       ctx,
		locch:     make(chan struct{}),
		locRecvch: make(chan Geolocation),
		quitch:    make(chan struct{}),
	}
}

// How to handle the geolocation
func (s *Server) Start() error {
	// Initialize the modem
	// Modem present, normal case geolocate using OpenCellId
	if s.m != nil {
		// Initialize the modem
		if err := s.m.Init(); err != nil {
			return err
		}

		locator := NewCellLocator(s.m, s.locRecvch, s.locch)

		// run the modem
		go s.m.Run()
		// run the location service
		go s.handleLocating(locator)
		log.Println("Starting Geolocation Server with Cellular Locator")
	} else {
		// Modem not present, fallback case geolocate using ip
		locator := NewLanLocator(s.locch, s.locRecvch)
		go s.handleLocating(locator)
		log.Println("Starting Geolocation Server with LAN Locator")
	}

	// Send first request right away
	s.locch <- struct{}{}

	// Ticker that delays the requests
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			// Instruct the locator to update the location
			s.locch <- struct{}{}
		case geo := <-s.locRecvch:
			// New geolocation received, do something with it, store it in db and map
			s.Location = geo
			log.Printf("New geolocation received: \n%+v\n", geo)
		case <-s.quitch:
			ticker.Stop()
			return nil
		}
	}
}

// Wrapper function to run the locator
func (s *Server) handleLocating(loc Locator) {
	// Run the locator
	loc.Run()
}
