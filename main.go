package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var GrometVersion string
var GitCommit string

const fsErrorCommandDefault = "/usr2/fs/bin/lgerr"

type fsErrorWriter struct {
	command string
}

func (f fsErrorWriter) Write(p []byte) (n int, err error) {
	cmd := exec.Command(f.command, "lg", "-1", string(p))
	err = cmd.Run()
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

type windstate struct {
	t         time.Time
	speed     float64
	direction float64
}

func (ws *windstate) marshal() string {
	if (ws.t == time.Time{}) {
		return ",,"
	}

	w := bytes.Buffer{}

	if math.IsNaN(ws.speed) {
		w.WriteString(",")
	} else {
		fmt.Fprintf(&w, "%.1f,", ws.speed)
	}

	if math.IsNaN(ws.direction) {
		w.WriteString(",")
	} else {
		fmt.Fprintf(&w, "%.1f,", ws.direction)
	}
	return w.String()
}

func openWindConn(addr string) <-chan windstate {
	s := windstate{}
	ws := make(chan windstate, 1)

	go func() {
	ConnLoop:
		for {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				log.Printf("error opening wind device connection: %s", err)
				time.Sleep(10 * time.Second)
				continue
			}

			for {
				b := bufio.NewReader(conn)
				resp, err := b.ReadString('\n')
				resp = strings.TrimSpace(resp)

				if resp == "Selected hunt group busy" {
					conn.Close()
					continue ConnLoop
				}

				if err == io.EOF {
					conn.Close()
					continue ConnLoop
				}

				if err != nil {
					log.Printf("error reading from wind device: %s", err)
					conn.Close()
					continue ConnLoop
				}

				if len(resp) <= 1 {
					log.Printf("error reading from wind device: device returned no data")
					conn.Close()
					continue ConnLoop
				}

				fields := strings.FieldsFunc(resp[1:], func(r rune) bool { return r == ',' })

				if len(fields) < 4 {
					log.Printf("error reading from wind: unexpected response %q", resp)
					conn.Close()
					continue ConnLoop
				}

				s.t = time.Now()
				s.speed, err = strconv.ParseFloat(fields[3], 64)
				if err != nil {
					s.speed = math.NaN()
					log.Printf("error decoding message wind device: wind speed given as %q", fields[3])
				}

				s.direction, err = strconv.ParseFloat(fields[1], 64)
				if err != nil {
					s.direction = math.NaN()
					log.Printf("error decoding message wind device: wind direction given as %q", fields[1])
				}
				ws <- s
			}
		}
	}()

	return ws
}

type metstate struct {
	t           time.Time
	pressure    float64
	temperature float64
	humidity    float64

	fanStatus bool
	fanRate   float64
}

func (ms *metstate) marshal() string {
	if (ms.t == time.Time{}) {
		return ",,,"
	}

	w := bytes.Buffer{}

	if math.IsNaN(ms.temperature) {
		w.WriteString(",")
	} else {
		fmt.Fprintf(&w, "%.1f,", ms.temperature)
	}

	if math.IsNaN(ms.pressure) {
		w.WriteString(",")
	} else {
		// NOTE: met server output expected to be in HPa
		fmt.Fprintf(&w, "%.1f,", 1e3*ms.pressure)
	}

	if math.IsNaN(ms.humidity) {
		w.WriteString(",")
	} else {
		fmt.Fprintf(&w, "%.1f,", ms.humidity)
	}
	return w.String()
}

const (
	MET4AFanStatusCommand = "*0100FS\r\n"
	MET4AFanStatusResp    = "*0001FS=%v"

	MET4AFanRateCommand = "*0100FR\r\n"
	MET4AFanRateResp    = "*0001FR=%v"

	MET4QueryCommand = "*0100P9\r\n"
)

func openMetConn(c *MetConfig, a AlertsConfig) <-chan metstate {
	s := metstate{}

	ms := make(chan metstate, 1)
	go func() {
	Conn:
		for {
			conn, err := net.Dial("tcp", c.Address)
			if err != nil {
				log.Printf("error opening MET device connection: %s", err)
				time.Sleep(10 * time.Second)
				continue
			}

			b := bufio.NewReader(conn)
			for {
				fmt.Fprintf(conn, MET4QueryCommand)
				resp, err := b.ReadString('\n')
				resp = strings.TrimSpace(resp)

				if resp == "Selected hunt group busy" {
					conn.Close()
					continue Conn
				}

				if err == io.EOF {
					conn.Close()
					continue Conn
				}

				if err != nil {
					log.Printf("error while reading from met device: %s", err)
					conn.Close()
					time.Sleep(10 * time.Second)
					continue Conn
				}

				fields := strings.FieldsFunc(resp, func(r rune) bool { return r == ',' })

				if len(fields) < 11 {
					log.Printf("received bad response from met device %q", resp)
					conn.Close()
					time.Sleep(10 * time.Second)
					continue Conn
				}

				s.t = time.Now()
				s.pressure, err = strconv.ParseFloat(fields[2], 64)
				if err != nil {
					s.pressure = math.NaN()
				}

				s.temperature, err = strconv.ParseFloat(fields[6], 64)
				if err != nil {
					s.temperature = math.NaN()
				}

				s.humidity, err = strconv.ParseFloat(fields[10], 64)
				if err != nil {
					s.humidity = math.NaN()
				}

				if c.Type != "MET4A" {
					ms <- s
					continue
				}

				fmt.Fprintf(conn, MET4AFanStatusCommand)
				resp, err = b.ReadString('\n')
				resp = strings.TrimSpace(resp)

				if resp == "Selected hunt group busy" {
					conn.Close()
					time.Sleep(2 * time.Second)
					continue Conn
				}

				if err == io.EOF {
					log.Printf("error while reading fan status from met device: unexpected end of input")
					conn.Close()
					continue Conn
				}

				if err != nil {
					log.Printf("error while reading fan status from met device: %s", err)
					conn.Close()
					time.Sleep(10 * time.Second)
					continue Conn
				}

				_, err = fmt.Sscanf(resp, MET4AFanStatusResp, &s.fanStatus)
				if err != nil {
					log.Printf("error parsing fan status from met device: %s in response %q", err, resp)
					conn.Close()
					time.Sleep(10 * time.Second)
					continue Conn
				}

				fmt.Fprintf(conn, MET4AFanRateCommand)
				resp, err = b.ReadString('\n')
				resp = strings.TrimSpace(resp)

				if resp == "Selected hunt group busy" {
					conn.Close()
					time.Sleep(2 * time.Second)
					continue Conn
				}

				if err == io.EOF {
					log.Printf("error while reading fan rate from met device: unexpected end of input")
					conn.Close()
					time.Sleep(10 * time.Second)
					continue Conn
				}

				if err != nil {
					log.Printf("error while reading fan rate from met device: %s", err)
					conn.Close()
					time.Sleep(10 * time.Second)
					continue Conn
				}

				_, err = fmt.Sscanf(resp, MET4AFanRateResp, &s.fanRate)
				if err != nil {
					log.Printf("error parsing fan rate from met device: %s in response %q", err, resp)
					conn.Close()
					time.Sleep(10 * time.Second)
					continue Conn
				}

				ms <- s
			}
		}
	}()
	return ms
}

type MetConfig struct {
	Address string
	Type    string
}

type WindConfig struct {
	Address string
}

type AlertsConfig struct {
	Mask []string
	Fs   struct {
		Enabled bool
		Path    string
	}
	Email struct {
		Enabled   bool
		Addresses []string
	}
}

type config struct {
	ListenAddress string `mapstructure:"listen_address"`
	Met           *MetConfig
	Wind          *WindConfig
	Alerts        AlertsConfig
}

const metTimeout = 5 * time.Minute
const windTimeout = 60 * time.Second

func openListener(address string) chan net.Conn {
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}

	conns := make(chan net.Conn)
	go func() {
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Fatalf("unable to listen on addres %q: %s", address, err)
			}
			conns <- conn
		}
	}()

	return conns

}

func Restart() {
	err := syscall.Exec(os.Args[0], os.Args, os.Environ())
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintln(os.Stderr, "version:", GrometVersion)
			fmt.Fprintln(os.Stderr, "commit:", GitCommit)
			panic(err)
		}
	}()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Println("version:", GrometVersion)
			fmt.Println("commit:", GitCommit)
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
			os.Exit(1)
		}
	}

	log.SetPrefix("gromet: ")
	log.SetFlags(0)

	log.Printf("starting...")

	viper.SetConfigName("gromet")
	viper.AddConfigPath("/etc/gromet/")
	viper.AddConfigPath("$HOME/.gromet")
	viper.AddConfigPath("/usr2/control")
	viper.AddConfigPath(".")

	viper.SetDefault("listen_address", "127.0.0.1:50001")

	viper.SetDefault("alerts.fs.enabled", true)
	viper.SetDefault("alerts.fs.path", fsErrorCommandDefault)
	viper.SetDefault("alerts.mask", []string{})
	viper.SetDefault("alerts.email.enabled", true)
	viper.SetDefault("alerts.email.addresses", []string{"oper"})

	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		log.Println("configuration changed, restarting...")
		Restart()
	})

	log.Printf("loading configuration")

	err := viper.ReadInConfig()

	if err != nil {
		log.Fatalf("error loading config file: %s \n", err)
	}

	var c config

	err = viper.Unmarshal(&c)
	if err != nil {
		log.Fatalf("error parsing config file: %s \n", err)
	}

	log.Printf("listening on %s \n", c.ListenAddress)
	conns := openListener(c.ListenAddress)

	logOutputs := []io.Writer{os.Stderr}
	if c.Alerts.Fs.Enabled {
		logOutputs = append(logOutputs, fsErrorWriter{fsErrorCommandDefault})
	}
	log.SetOutput(io.MultiWriter(logOutputs...))

	var metStates <-chan metstate
	metTimer := new(time.Timer)
	var metTimedOut bool
	if c.Met != nil {
		metStates = openMetConn(c.Met, c.Alerts)
		metTimer = time.NewTimer(metTimeout)
	}

	var windStates <-chan windstate
	windTimer := new(time.Timer)
	var windTimedOut bool
	if c.Wind != nil {
		windStates = openWindConn(c.Wind.Address)
		windTimer = time.NewTimer(windTimeout)
	}

	var met metstate
	var wind windstate

	for {
		select {
		case <-metTimer.C:
			met = metstate{}
			log.Println("reading from met device timed out")
			metTimedOut = true
		case met = <-metStates:
			if metTimedOut {
				log.Println("met device communication restored")
				metTimedOut = false
			} else {
				if !metTimer.Stop() {
					<-metTimer.C
				}
			}
			metTimer.Reset(metTimeout)

		case <-windTimer.C:
			wind = windstate{}
			log.Println("reading from wind device timed out")
			windTimedOut = true
		case wind = <-windStates:
			windTimer.Reset(windTimeout)
			if windTimedOut {
				log.Println("wind device communication restored")
				windTimedOut = false
			} else {
				if !windTimer.Stop() {
					<-windTimer.C
				}
			}
			windTimer.Reset(windTimeout)

		case conn := <-conns:
			w := bufio.NewWriter(conn)
			w.WriteString(met.marshal())
			w.WriteString(wind.marshal())
			w.Flush()
			conn.Close()
		}
	}
}
