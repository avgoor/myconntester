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

func main() {
	mHostName := flag.String("hostname", "127.0.0.1", "hostname/IP of MySQL server")
	mPort := flag.String("port", "3306", "MySQL server port")
	mDatabase := flag.String("database", "mysql", "MySQL database to connect")
	mUsername := flag.String("username", "mysql", "MySQL username")
	mPassword := flag.String("password", "", "MySQL password")
	mCaPemFile := flag.String("ssl-ca", "", "SSL CA file")
	mCertPemFile := flag.String("ssl-cert", "", "SSL certificate file")
	mKeyPemFile := flag.String("ssl-key", "", "SSL certificate key file")
	mInsecure := flag.Bool("insecure", false, "Skip SSL checks")

	flag.Parse()
	var tlsConfig *tls.Config
	if *mCertPemFile != "" && *mKeyPemFile != "" && *mCaPemFile != "" {
		mCaPem, err := ioutil.ReadFile(*mCaPemFile)
		if err != nil {
			log.Fatal(err)
		}
		mCertPem, err := ioutil.ReadFile(*mCertPemFile)
		if err != nil {
			log.Fatal(err)
		}
		mKeyPem, err := ioutil.ReadFile(*mKeyPemFile)
		if err != nil {
			log.Fatal(err)
		}
		tlsConfig = client.NewClientTLSConfig(mCaPem, mCertPem, mKeyPem,
			*mInsecure, *mHostName)
	}

	conn, err := client.Connect(strings.Join([]string{*mHostName, *mPort}, ":"),
		*mUsername, *mPassword, *mDatabase, func(c *client.Conn) { c.SetTLSConfig(tlsConfig) })
	if err != nil {
		log.Fatal(err)
	}
	r, err := conn.Execute("SHOW STATUS;")
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range r.Values {
		fmt.Printf("%s:%s\n", row[0].AsString(), row[1].AsString())
	}
}
