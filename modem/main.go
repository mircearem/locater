package modem

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

var (
	COMMAND   = "/etc/config-tools/config_mdmd-ng"
	MDMD_ARGS = []string{"-m", "get", "json"}
	CONN_ARGS = []string{"-n", "get", "json"}
	WDS_ARGS  = []string{"-w", "get", "json"}
)

// Response from the WAGO modem API
type conn struct {
	Cid    string `json:"cid"`
	Lac    string `json:"lac"`
	Mccmnc string `json:"mccmnc"`
}

type NetworkIdentifier struct {
	Mnc int
	Mcc int
	Cid int
	Lac int
}

type net struct {
	FallbackToAuto      string `json:"fallback_to_auto"`
	Operator            string `json:"operator"`
	OperatorIdentifier  string `json:"operator_identifier"`
	OperatorShort       string `json:"operator_short"`
	RegistrationMode    string `json:"registration_mode"`
	SignalRssi          int    `json:"signal_rssi"`
	SignalStrength      int    `json:"signal_strength"`
	State               string `json:"state"`
	Technology          string `json:"technology"`
	TechnologySelection string `json:"technology_selection"`
}

type info struct {
	Imei         string `json:"imei"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	State        string `json:"state"`
	Version      string `json:"version"`
}

type network struct {
	Cid                 int    `json:"cid"`
	FallbackToAuto      string `json:"fallback_to_auto"`
	Lac                 int    `json:"lac"`
	Mcc                 int    `json:"mcc"`
	Mnc                 int    `json:"mnc"`
	Operator            string `json:"operator"`
	OperatorIdentifier  string `json:"operator_identifier"`
	OperatorShort       string `json:"operator_short"`
	RegistrationMode    string `json:"registration_mode"`
	SignalRssi          int    `json:"signal_rssi"`
	SignalStrength      int    `json:"signal_strength"`
	State               string `json:"state"`
	Technology          string `json:"technology"`
	TechnologySelection string `json:"technology_selection"`
}

// Wireless data service
type wds struct {
	Apn    string `json:"apn"`
	IP     string `json:"ip"`
	State  string `json:"state"`
	Status string `json:"status"`
}

type Modem struct {
	Info     info    `json:"info"`
	Wireless wds     `json:"wds"`
	Network  network `json:"network"`
	ctx      context.Context
}

func NewModem(ctx context.Context) (*Modem, error) {
	// Check if a modem is installed on the system
	if _, err := os.Stat(COMMAND); errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return &Modem{
		ctx: ctx,
	}, nil
}

// Function to initialize the and get all the information
func (m *Modem) Init() error {
	errch := make(chan error, 3)

	var wg sync.WaitGroup
	for i := 0; i < 3; i += 1 {
		wg.Add(1)
		go func(x int) {
			switch x {
			case 0:
				err := m.mdmdInfo()
				errch <- err
			case 1:
				err := m.networkInfo()
				errch <- err
			case 2:
				err := m.wdsInfo()
				errch <- err
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	close(errch)

	// Check for errors
	for err := range errch {
		if err != nil {
			return err
		}
	}

	return nil
}

// Run the modem update
func (m *Modem) Run() {
	// Move the stuff below in a netork update function
	ticker := time.NewTicker(5 * time.Minute)

	var wg sync.WaitGroup

	for {
		<-ticker.C
		for i := 0; i < 2; i += 1 {
			wg.Add(1)
			go func(x int) {
				switch x {
				case 0:
					_ = m.networkInfo()
				case 1:
					_ = m.wdsInfo()
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
	}
}

// Read modem information -> stays the same, except for the state
func (m *Modem) mdmdInfo() error {
	// Execute the command
	bytes, err := exec.CommandContext(m.ctx, COMMAND, MDMD_ARGS...).Output()
	// Modem not present, set error state
	if err != nil {
		err := errors.New("NOT PRESENT")
		return err
	}

	err = json.Unmarshal(bytes, &m)
	if err != nil {
		err := errors.New("INFO DECODE ERR")
		return err
	}

	return nil
}

// Read information about the wireless data service -> can change
func (m *Modem) wdsInfo() error {
	// Execute the command
	bytes, err := exec.CommandContext(m.ctx, COMMAND, WDS_ARGS...).Output()
	// Modem not present, set error state
	if err != nil {
		err := errors.New("NOT PRESENT")
		return err
	}

	var wds wds
	err = json.Unmarshal(bytes, &wds)
	if err != nil {
		err := errors.New("INFO DECODE ERR")
		return err
	}

	m.Wireless = wds

	return nil
}

// Update information regarding the connection -> can change
func (m *Modem) networkInfo() error {
	bytes, err := exec.CommandContext(m.ctx, COMMAND, CONN_ARGS...).Output()
	if err != nil {
		err := errors.New("CONN READ ERR")
		return err
	}

	var conn conn
	err = json.Unmarshal(bytes, &conn)
	if err != nil {
		err := errors.New("CONN DECODE ERR")
		return err
	}

	var net net
	err = json.Unmarshal(bytes, &net)
	if err != nil {
		err := errors.New("NETWORK DECODE ERR")
		return err
	}

	// Reformat the cid, lac, mcc and mnc to int
	cid, _ := strconv.Atoi(conn.Cid)
	m.Network.Cid = cid
	lac, _ := strconv.Atoi(conn.Lac)
	m.Network.Lac = lac
	mcc, _ := strconv.Atoi(conn.Mccmnc[:3])
	m.Network.Mcc = mcc
	mnc, _ := strconv.Atoi(conn.Mccmnc[3:])
	m.Network.Mnc = mnc
	// Format network information
	m.Network.FallbackToAuto = net.FallbackToAuto
	m.Network.Operator = net.Operator
	m.Network.OperatorIdentifier = net.OperatorIdentifier
	m.Network.OperatorShort = net.OperatorShort
	m.Network.RegistrationMode = net.RegistrationMode
	m.Network.SignalRssi = net.SignalRssi
	m.Network.SignalStrength = net.SignalStrength
	m.Network.State = net.State
	m.Network.Technology = net.Technology
	m.Network.TechnologySelection = net.TechnologySelection

	return nil
}
