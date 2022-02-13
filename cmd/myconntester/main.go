package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"syscall"
	"time"

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
	log.SetFlags(log.Lshortfile)
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

	// adjust fd limits
	var rLim syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLim)
	if err != nil {
		log.Fatal(err)
	}
	needed := uint64(cfg.Preload + cfg.TestCount + 1024)
	if rLim.Cur < needed {
		fmt.Printf("Adjusting NOFILE limit from %d to %d\n", rLim.Cur, needed)
		rLim.Cur = needed
		rLim.Max = needed
		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLim)
		if err != nil {
			log.Fatal(err)
		}
	}

	connector := func() *client.Conn {
		conn, err := client.Connect(strings.Join([]string{cfg.Hostname, cfg.Port}, ":"),
			cfg.Username, cfg.Password, cfg.Database, func(c *client.Conn) { c.SetTLSConfig(tlsConfig) })
		if err != nil {
			log.Fatal(err)
		}
		result, err := conn.Execute("SELECT 1;")
		result.Close()

		if err != nil {
			log.Fatal(err)
		}
		return conn
	}
	if cfg.Preload > 0 {
		fmt.Println("Precreating connections")
		pConnections := make([]*client.Conn, cfg.Preload)
		for i := 0; i < cfg.Preload; i++ {
			pConnections[i] = connector()
		}
		defer func() {
			for _, c := range pConnections {
				c.Close()
			}
			fmt.Printf("Closed %d precreated connections.\n", len(pConnections))
		}()
		fmt.Printf("Precreated: %d connections\n", len(pConnections))
	}
	fmt.Println("Starting test")
	var wg sync.WaitGroup
	tConnections := make([]*client.Conn, cfg.TestCount)
	bucket := make(chan struct{}, cfg.Concurrency)

	startTime := time.Now()

	for i := 0; i < cfg.TestCount; i++ {
		bucket <- struct{}{}
		wg.Add(1)
		go func(c int) {
			tConnections[c] = connector()
			<-bucket
			wg.Done()
		}(i)
	}
	defer func() {
		for _, c := range tConnections {
			c.Close()
		}
		fmt.Printf("Closed %d connections.\n", len(tConnections))
	}()
	wg.Wait()
	elapsedTime := time.Since(startTime)
	fmt.Printf("%d connections with concurrency %d created in %v\n", cfg.TestCount, cfg.Concurrency, elapsedTime)
	fmt.Printf("1 connection took on average %v to establish\n", elapsedTime/time.Duration(cfg.TestCount))
}
