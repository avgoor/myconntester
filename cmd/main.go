package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/go-mysql-org/go-mysql/client"
)

type configOpt struct {
	Hostname    string
	Port        string
	Database    string
	Username    string
	Password    string
	CaPemFile   string
	CertPemFile string
	KeyPemFile  string
	Insecure    bool
	Concurrency int
	Preload     int
	TestCount   int
}

func main() {
	var cfg configOpt

	flag.StringVar(&cfg.Hostname, "H", "127.0.0.1", "hostname/IP of MySQL server")
	flag.StringVar(&cfg.Port, "P", "3306", "MySQL server port")
	flag.StringVar(&cfg.Database, "d", "mysql", "MySQL database to connect")
	flag.StringVar(&cfg.Username, "u", "mysql", "MySQL username")
	flag.StringVar(&cfg.Password, "p", "", "MySQL password")
	flag.StringVar(&cfg.CaPemFile, "ssl-ca", "", "SSL CA file")
	flag.StringVar(&cfg.CertPemFile, "ssl-cert", "", "SSL certificate file")
	flag.StringVar(&cfg.KeyPemFile, "ssl-key", "", "SSL certificate key file")
	flag.BoolVar(&cfg.Insecure, "i", false, "Skip SSL checks")
	flag.IntVar(&cfg.Concurrency, "c", 10, "Concurrency")
	flag.IntVar(&cfg.Preload, "l", 0, "Create n connections before real testing")
	flag.IntVar(&cfg.TestCount, "t", 300, "Measure creation of n connections")
	flag.Parse()

	var tlsConfig *tls.Config

	fmt.Println("Started with opts: ", cfg)

	if cfg.CertPemFile != "" && cfg.KeyPemFile != "" && cfg.CaPemFile != "" {
		mCaPem, err := ioutil.ReadFile(cfg.CaPemFile)
		if err != nil {
			log.Fatal(err)
		}
		mCertPem, err := ioutil.ReadFile(cfg.CertPemFile)
		if err != nil {
			log.Fatal(err)
		}
		mKeyPem, err := ioutil.ReadFile(cfg.KeyPemFile)
		if err != nil {
			log.Fatal(err)
		}
		tlsConfig = client.NewClientTLSConfig(mCaPem, mCertPem, mKeyPem,
			cfg.Insecure, cfg.Hostname)
	}

	connector := func() *client.Conn {
		conn, err := client.Connect(strings.Join([]string{cfg.Hostname, cfg.Port}, ":"),
			cfg.Username, cfg.Password, cfg.Database, func(c *client.Conn) { c.SetTLSConfig(tlsConfig) })
		if err != nil {
			log.Fatal(err)
		}
		_, err = conn.Execute("SELECT 1;")
		if err != nil {
			log.Fatal(err)
		}
		return conn
	}
	if cfg.Preload > 0 {
		var connections []*client.Conn
		for i := 0; i < cfg.Preload; i++ {
			connections = append(connections, connector())
		}
		defer func() {
			for _, c := range connections {
				c.Close()
			}
			fmt.Printf("Closed %d precreated connections.\n", len(connections))
		}()
		fmt.Println("Precreated: ", connections)
	}
}
