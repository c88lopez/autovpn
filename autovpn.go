package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

const sourceURL = "http://www.vpngate.net/api/iphone/"

func main() {
	chosenCountry := "US"
	if len(os.Args) > 1 && len(os.Args[1]) == 2 {
		chosenCountry = os.Args[1]
	}

	log.Println("[autovpn] Getting server list...")
	response, err := http.Get(sourceURL)
	check(err)
	defer response.Body.Close()

	csvFile, err := ioutil.ReadAll(response.Body)
	check(err)

	log.Println("[autovpn] Parsing response...")
	log.Printf("[autovpn] Looking for %s...\n", chosenCountry)

	for i, line := range strings.Split(string(csvFile), "\n") {
		if i <= 1 {
			continue
		}

		splits := strings.Split(line, ",")
		if len(splits) < 15 {
			break
		}

		country := splits[6]
		conf, err := base64.StdEncoding.DecodeString(splits[14])
		if err != nil || chosenCountry != country {
			continue
		}

		writeConfFile(conf)

		log.Println("[autovpn] Running openvpn...")

		cmd := exec.Command("sudo", "openvpn", "/tmp/openvpnconf")
		cmd.Stdout = os.Stdout

		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			cmd.Process.Kill()
		}()

		cmd.Start()
		cmd.Wait()

		checkTryAnother()
	}

	log.Println("[autovpn] No more vpns to connect.")
}

func writeConfFile(c []byte) {
	log.Println("[autovpn] Writing config file...")
	f, err := os.Create("/tmp/openvpnconf")
	check(err)
	defer f.Close()

	_, err = f.Write(c)
	check(err)
}

func checkTryAnother() {
	fmt.Print("[autovpn] Try another VPN? (y/n) ")

	var input string
	fmt.Scanln(&input)
	if strings.ToLower(input) == "n" {
		fmt.Println("[autovpn] Bye!")
		os.Exit(0)
	}
}
