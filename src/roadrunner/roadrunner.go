// Go IMAP roadrunner
// Simple IMAP benchmarking tool

// Apache License
// Author: Ben Duncan, Atmail CTO
// (C) Atmail Pty Ltd

package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mxk/go-imap/imap"
)

func main() {

	start := time.Now()

	// Required CLI flags to connect to the remote IMAP server
	user := flag.String("user", "", "Username to authenticate (required)")
	pass := flag.String("pass", "", "Password to authenticate (required)")
	server := flag.String("server", "", "Remote IMAP server (required)")

	// Optional flags
	folder := flag.String("folder", "Inbox", "Folder to select")
	cycle := flag.Int("cycle", 3, "Number of times to cycle")
	csv := flag.Bool("csv", false, "Flag for CSV output")

	tls := flag.Bool("tls", false, "Flag for connecting via TLS/SSL")

	flag.Parse()

	// Validate the user CLI input
	if *user == "" {
		fmt.Println("User missing from arguments")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Require a password
	if *pass == "" {
		fmt.Println("Password missing from arguments")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Require a server to be specified
	if *server == "" {
		fmt.Println("Server missing from arguments")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// If not exporting in CSV format, display our header banner
	if *csv == false {
		fmt.Println("Go IMAP Roadrunner at your service")
	} else {
		fmt.Println("\"Cycle\", \"Message ID\", \"Message in Bytes\", \"Execution time in secs\", \"IMAP Command\"")
	}

	// Run the IMAP sequence X times, useful for benchmarking the same request multiple times.
	for i := 1; i <= *cycle; i++ {

		if *csv == false {
			fmt.Println("Launch cycle ", i)
		}

		// Run the test
		BenchmarkIMAP(*folder, *user, *pass, *server, *csv, *tls, i)
	}

	// Output the total run-time, if we are not exporting to CSV
	if *csv == false {
		elapsed := time.Since(start)
		fmt.Println("Total run time => ", elapsed)
	}

}

func BenchmarkIMAP(folder, user, pass, server string, csv bool, tlsimap bool, cycle int) {

	// Change the port depending on TLS enabled
	if tlsimap == false {
		s := []string{server, ":143"}
		server = strings.Join(s, "")
	} else {
		s := []string{server, ":993"}
		server = strings.Join(s, "")
	}

	// First, connect to the specified IMAP server
	c, total_msgs, err := ConnectIMAP(folder, user, pass, server, tlsimap)

	// Can we connect?
	if err != nil {
		log.Fatal("Could not connect to IMAP: ", err)
	}

	// Require messages to exist in the folder, exit otherwise.
	if c == nil {
		log.Fatal("Folder count for ", folder, "is empty. Please check the mailbox contains messages")
	}

	// If not using CSV, export the number of messages contained in the folder.
	if csv == false {
		fmt.Println("Server responded with =>", total_msgs, "total messages in", folder)
	}

	// Specify the headers required
	// TODO: Specify these on the CLI as an optional argument
	items := []string{
		"INTERNALDATE",
		"FLAGS",
		"RFC822.SIZE",
		"BODY.PEEK[]",
	}

	// Loop through each message, request the entire message and time the response
	for i := 1; i <= int(total_msgs); i++ {
		set, _ := imap.NewSeqSet("")
		set.AddNum(uint32(i))
		FetchMail(c, set, csv, cycle, items)
	}

	// TODO: Add CLI args to do entire "range" as another benchmark
	// Fetch 1:X messages in one hit, vs individually.
	set, _ := imap.NewSeqSet("")

	// Loop through each message, request the entire message and time the response
	for i := 1; i <= int(total_msgs); i++ {
		set.AddNum(uint32(i))
	}

	// TODO: Add debug flag
	//fmt.Println("Fetching entire range =>", set)

	FetchMail(c, set, csv, cycle, items)

	// Fetch only message headers, no body
	items = []string{
		"INTERNALDATE",
		"FLAGS",
		"RFC822.SIZE",
		"RFC822.HEADER",
	}

	//fmt.Println("Fetching just messasge headers =>", set)
	FetchMail(c, set, csv, cycle, items)

	//fmt.Println("Search performance =>", set)

	// Search performance, subject
	SearchMail(c, set, csv, cycle, "SUBJECT", "ben")

	// Search performance, body
	// TODO: Add search key/value terms on CLI
	SearchMail(c, set, csv, cycle, "BODY", "nova")

	// Logout once complete
	c.Logout(0)

}

func dialIMAP(hostname string, tlsimap bool) (client *imap.Client, err error) {

	var c *imap.Client
	var e error

	if tlsimap == false {
		c, e = imap.Dial(hostname)

	} else {
		c, e = imap.DialTLS(hostname, &tls.Config{InsecureSkipVerify: true})
	}

	// Add for debug flag
	//c.SetLogMask(imap.LogAll)

	return c, e

}

func ConnectIMAP(folder, user, passwd, server string, tlsimap bool) (client *imap.Client, total uint32, err error) {

	c, err := dialIMAP(server, tlsimap)

	// If the host does not connect, return an error
	if err != nil {
		return c, 0, err
	}

	if c.State() == imap.Login {
		c.Login(user, passwd)
	}

	// Confirm the username/password combination is successful
	if c.State() != imap.Auth {
		log.Println("Authentication failed")
		return c, 0, errors.New("Cannot authenticate: " + user)
	}

	// Select the specified folder, used for fetch requests next
	c.Select(folder, true)

	return c, c.Mailbox.Messages, nil
}

func FetchMail(c *imap.Client, seq *imap.SeqSet, csv bool, cycle int, items []string) (total uint32) {

	start := time.Now()

	// TODO: Wait, sync, if no wait, polls until buffer used?
	cmd, err := imap.Wait(c.Fetch(seq, items...))

	// Return if there is an error from the remote IMAP (e.g message no longer exists)
	if err != nil {
		log.Println("Message", seq, "could not be retrieved:( ", err, ")")
		return 0
	}

	// Read the raw IMAP response into a buffer
	rsp := cmd.Data[0]
	buf := new(bytes.Buffer)

	for _, l := range rsp.Literals {
		l.WriteTo(buf)
	}

	Content := buf.String()
	size := len(Content)
	elapsed := time.Since(start)

	// Output the stats on the message ID, size and how long the request took.
	if csv == true {
		fmt.Println(cycle, ",", seq, ",", size, ",", elapsed.Seconds(), ",", cmd)
	} else {
		fmt.Println(cmd, " => IMAP reply =>", size, "bytes", "( received in ", elapsed.Seconds(), ")")

	}

	// TODO: Return the stats above, let the parent function handle the output.
	return 0
}

func SearchMail(c *imap.Client, seq *imap.SeqSet, csv bool, cycle int, field string, value string) (total uint32) {

	start := time.Now()

	cmd, err := imap.Wait(c.Search(field, value))

	var rsp *imap.Response
	if cmd == nil {
		panic(err)
	} else if err == nil {
		rsp, err = cmd.Result(imap.OK)
	}

	if err != nil {
		panic(err)
	}

	// Read the raw IMAP response into a buffer

	rsp = cmd.Data[0]
	buf := new(bytes.Buffer)

	for _, l := range rsp.Literals {
		l.WriteTo(buf)
	}

	Content := buf.String()
	size := len(Content)
	elapsed := time.Since(start)

	// Output the stats on the message ID, size and how long the request took.
	if csv == true {
		fmt.Println(cycle, ",", seq, ",", size, ",", elapsed.Seconds(), ",", cmd)
	} else {
		fmt.Println(cmd, "=> IMAP reply =>", size, "bytes", "( received in ", elapsed.Seconds(), ")")

	}

	// TODO: Return the stats above, let the parent function handle the output.
	return 0
}
